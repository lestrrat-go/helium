<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  xmlns:err="http://www.w3.org/2005/xqt-errors" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:f="http://www.w3.org/xslt30tests/functions" exclude-result-prefixes="map xs err f">

  <xsl:variable name="RUN" select="true()" static="yes"/>
  <xsl:strip-space elements="*"/>


  <xsl:function name="f:count-descendants-001" as="xs:integer" streamability="absorbing">
    <xsl:param name="input" as="node()*"/>
    <xsl:sequence select="count($input!descendant::*)"/>
  </xsl:function>

  <xsl:template name="t-001" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(/BOOKLIST/BOOKS/ITEM) ! f:count-descendants-001(.)"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <xsl:template name="t-002" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="f:count-descendants-001(/BOOKLIST/BOOKS/ITEM)"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <xsl:function name="f:count-descendants-003" as="xs:integer" streamability="absorbing">
    <xsl:param name="input" as="node()*"/>
    <xsl:sequence select="count($input/descendant::*)"/>
  </xsl:function>
  
  <xsl:template name="t-003" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(/BOOKLIST/BOOKS/ITEM) ! f:count-descendants-003(.)"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <xsl:template name="t-004" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="f:count-descendants-003(/BOOKLIST/BOOKS/ITEM)"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- recursive absorbing function -->
  
  <xsl:function name="f:count-descendants-or-self-005" as="xs:integer" streamability="absorbing">
    <xsl:param name="input" as="node()"/>
    <xsl:sequence select="1 + sum($input/* ! f:count-descendants-or-self-005(.))"/>
  </xsl:function>
  
  <xsl:template name="t-005" use-when="true() or $RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(/BOOKLIST/BOOKS/ITEM) ! f:count-descendants-or-self-005(.)"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Call absorbing function with grounded input -->
  
  <xsl:template name="t-006" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="copy-of(/BOOKLIST/BOOKS/ITEM) ! f:count-descendants-or-self-005(.)"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- two references to the variable allowed if you're careful -->
  
  <xsl:function name="f:f-007" as="xs:string" streamability="absorbing" use-when="$RUN">
    <xsl:param name="input" as="node()"/>
    <xsl:sequence select="namespace-uri($input) || '#' || local-name($input) || '?' || exists($input/*)"/>
  </xsl:function>
  
  <xsl:template name="t-007" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(/BOOKLIST/BOOKS/ITEM) ! f:f-007(.)"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Invoke streamable template from streamable function -->
  
  <xsl:mode name="m-008" streamable="yes"/>
  
  <xsl:template match="ITEM" mode="m-008">
    <xsl:sequence select="string(PRICE)"/>
  </xsl:template>
  
  <xsl:function name="f:f-008" as="element(data)" streamability="absorbing" use-when="$RUN">
    <xsl:param name="input" as="node()*"/>
    <data>
      <xsl:apply-templates select="$input" mode="m-008"/>
    </data>
  </xsl:function>
  
  <xsl:template name="t-008" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:copy-of select="f:f-008(/BOOKLIST/BOOKS/ITEM)"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- recursive absorbing function -->
  
  <xsl:function name="f:count-descendants-009" as="xs:integer" streamability="absorbing" use-when="$RUN">
    <xsl:param name="input" as="node()"/>
    <!--<xsl:sequence select="if (has-children($input)) then sum($input/* ! (1 + f:count-descendants-009(.))) else 0"/>-->
    <xsl:sequence select="sum($input/* ! (1 + f:count-descendants-009(.)))"/>
  </xsl:function>
  
  <xsl:template name="t-009" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(/BOOKLIST/BOOKS/ITEM) ! f:count-descendants-009(.)"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- recursive absorbing function using has-children() -->
  
  <xsl:function name="f:count-descendants-010" as="xs:integer" streamability="absorbing" use-when="$RUN">
    <xsl:param name="input" as="node()?"/>
    <xsl:sequence select="if (has-children($input)) then sum($input/* ! (1 + f:count-descendants-010(.))) else 0"/>
  </xsl:function>
  
  <xsl:template name="t-010" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(/BOOKLIST/BOOKS/ITEM) ! f:count-descendants-010(.)"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Call streaming function supplying an empty sequence -->
  
  <xsl:template name="t-011" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="f:count-descendants-001(/BOOKLIST/MAGAZINES)"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Call singleton absorbing function supplying an empty sequence -->
  
  <xsl:template name="t-012" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="f:count-descendants-010(/BOOKLIST/MAGAZINES)"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Call absorbing function supplying grounded but consuming sequence -->
  
  <xsl:template name="t-013" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="f:count-descendants-001(copy-of(/BOOKLIST/BOOKS/ITEM))"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Call singleton absorbing function supplying grounded but consuming sequence -->
  
  <xsl:template name="t-014" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="/BOOKLIST/BOOKS/ITEM ! f:count-descendants-010(copy-of(TITLE))"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Call streamable function from streamable function -->
  
  <xsl:function name="f:f-015a" as="xs:integer*" streamability="absorbing" use-when="$RUN">
    <xsl:param name="input" as="node()*"/>
    <xsl:sequence select="$input ! f:f-015b(.)"/>
  </xsl:function>
  
  <xsl:function name="f:f-015b" as="xs:integer" streamability="absorbing" use-when="$RUN">
    <xsl:param name="input" as="node()"/>
    <xsl:sequence select="string-length($input)"/>
  </xsl:function>
  
  <xsl:template name="t-015" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="f:f-015a(/BOOKLIST/BOOKS/ITEM/TITLE)"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Call absorbing function from streamable template -->
  
  <xsl:mode name="m-016" streamable="true"/>
  
  <xsl:template match="TITLE" mode="m-016" use-when="$RUN">
    <xsl:sequence select="f:f-015b(.)"/>
  </xsl:template>
  
  <xsl:template match="TITLE/text()" mode="m-016" use-when="$RUN">
    <xsl:sequence select="f:f-015b(.)"/>
  </xsl:template>
  
  <xsl:template name="t-016" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:apply-templates select="/BOOKLIST/BOOKS/ITEM/TITLE" mode="m-016"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- Call absorbing function with text nodes -->
  
  <xsl:template name="t-017" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:apply-templates select="/BOOKLIST/BOOKS/ITEM/TITLE/text()" mode="m-016"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  
  

</xsl:stylesheet>
