<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="map xs">

  <xsl:variable name="RUN" select="true()" static="yes"/>
  <xsl:strip-space elements="*"/>

  <!-- Simple use of xsl:stream with "treat as" -->

  <xsl:template name="c-001" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(/*/* treat as element()*) ! name(.)"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- "treat as", filtered with motionless predicate -->

  <xsl:template name="c-002" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:copy-of select="(./BOOKLIST/BOOKS/*[@CAT = 'P']) treat as element(ITEM)*"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- "treat as" applied to ancestor nodes-->

  <xsl:template name="c-003" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of
          select="((BOOKLIST/BOOKS/ITEM[@CAT = 'MMP'] ! (ancestor::*)) treat as element(BOOKLIST)*) ! name()"
        />
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- "treat as" applied to a grounded consuming sequence -->

  <xsl:template name="c-004" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of
          select="(/BOOKLIST/BOOKS/ITEM/DIMENSIONS ! tokenize(., ' ')) treat as xs:string+"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- "treat as" applied to attributes of ancestor nodes-->

  <xsl:template name="c-005" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of
          select="(BOOKLIST/BOOKS/ITEM[@CAT = 'MMP'] ! (ancestor-or-self::*/@*)) treat as node()+"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- "treat as" applied to namespaces of ancestor nodes-->

  <xsl:template name="c-006" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of
          select="(BOOKLIST/BOOKS/ITEM[@CAT = 'MMP'] ! (ancestor-or-self::*/namespace::*)) treat as node()+"
        />
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- "treat as" with empty downwards selection-->

  <xsl:template name="c-007" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(BOOKS) treat as node()+"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- "treat as" with empty downwards selection with predicate-->

  <xsl:template name="c-008" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(BOOKS/BOOK[2]) treat as node()+"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- "treat as" with a striding(?) union -->

  <xsl:template name="c-009" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(BOOKLIST/BOOKS | BOOKLIST/CATEGORIES) treat as element(BOOKS)+"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- "treat as" with a crawling union -->

  <xsl:template name="c-010" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(.//* | .//text()) treat as text()*"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- simple motionless "treat as" -->

  <xsl:template name="c-011" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:if test="child::BOOKLIST">
          <xsl:value-of select="(1 to 10) treat as xs:anyAtomicType+"/>
        </xsl:if>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- "treat as" filtered grounded sequence -->

  <xsl:template name="c-012" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(remove(data(//DIMENSIONS/text()), 3)) treat as xs:untypedAtomic+"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- "treat as" filtered striding sequence -->

  <xsl:template name="c-013" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(remove(/BOOKLIST/BOOKS/ITEM, 3)) treat as element()+"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- "treat as" twice-filtered striding sequence -->

  <xsl:template name="c-014" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(remove(/BOOKLIST/BOOKS/ITEM, 3)[@CAT = 'MMP']) treat as node()?"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- "treat as" applied to a non-existent element -->

  <xsl:template name="c-015" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
      <out>
        <xsl:value-of select="(account/transaction/details) treat as node()+"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- "treat as" applied to an existent attribute (can exit early) -->

  <xsl:template name="c-016" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
      <out>
        <xsl:value-of select="(account/transaction/@value) treat as attribute()+"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- Test of xsl:stream with "treat as" and a boolean filter -->

  <xsl:template name="c-017" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
      <out>
        <xsl:value-of select="(account/transaction[@value &gt; 10000000]) treat as node()+"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- Test of xsl:stream with "treat as" and both a positional and a boolean filter -->

  <xsl:template name="c-018" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
      <out>
        <xsl:value-of
          select="(account/transaction[position() lt 20][@value &gt; 1000]) treat as node()+"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- "treat as" on a sequence of both streamed nodes and atomic values -->

  <xsl:template name="c-100" use-when="$RUN">
    <xsl:variable name="b" select="1[current-date() gt xs:date('1900-01-01')]"/>
    <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
      <out>
        <xsl:value-of select="(($b, account/transaction/dummy)) treat as node()+"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- "treat as" on a sequence of both streamed nodes and atomic values -->

  <xsl:template name="c-101" use-when="$RUN">
    <xsl:variable name="b" select="1[current-date() gt xs:date('1900-01-01')]"/>
    <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
      <out>
        <xsl:value-of select="((account/transaction/dummy, $b)) treat as node()*"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- "treat as" on a sequence of both streamed nodes and atomic values -->

  <xsl:template name="c-102" use-when="$RUN">
    <xsl:variable name="b" select="1[current-date() gt xs:date('1900-01-01')]"/>
    <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
      <out>
        <xsl:value-of select="((account/transaction, $b)) treat as node()*"/>
      </out>
    </xsl:source-document>
  </xsl:template>

  <!-- "treat as" on a sequence of both streamed nodes and atomic values -->

  <xsl:template name="c-103" use-when="$RUN">
    <xsl:variable name="b" select="1[current-date() gt xs:date('1900-01-01')]"/>
    <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
      <out>
        <xsl:value-of select="(($b, account/transaction)) treat as node()*"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- "treat as" applied to grounded element nodes -->
  
  <xsl:template name="c-104" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(outermost(//PRICE) ! parse-xml('&lt;p a=''3''>' || . || '&lt;/p>')/*) treat as element(p)*"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <xsl:template name="c-104a" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(outermost(//PRICE) ! parse-xml('&lt;p a=''3''>' || . || '&lt;/p>')/*) treat as attribute(p)*"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- "treat as" applied to grounded text nodes -->
  
  <xsl:template name="c-105" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(outermost(//PRICE) ! parse-xml('&lt;p a=''3''>' || . || '&lt;/p>')//text()) treat as text()+"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <xsl:template name="c-105a" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(outermost(//PRICE) ! parse-xml('&lt;p a=''3''>' || . || '&lt;/p>')//text()) treat as xs:string+"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- "treat as" applied to grounded attribute nodes -->
  
  <xsl:template name="c-106" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(outermost(//PRICE) ! parse-xml('&lt;p a=''3''>' || . || '&lt;/p>')//@a) treat as attribute(a)*"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <xsl:template name="c-106a" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(outermost(//PRICE) ! parse-xml('&lt;p a=''3''>' || . || '&lt;/p>')//@a) treat as element(a)*"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <xsl:function name="Q{f}attribute" as="attribute()">
    <xsl:param name="name" as="xs:string"/>
    <xsl:param name="value" as="xs:string"/>
    <xsl:attribute name="{$name}" select="$value"/>
  </xsl:function>
  
  <xsl:function name="Q{f}element" as="element()">
    <xsl:param name="name" as="xs:string"/>
    <xsl:param name="value" as="xs:string"/>
    <xsl:element name="{$name}">
      <xsl:attribute name="x" select="'y'"/>
      <xsl:value-of select="$value"/>
    </xsl:element>
  </xsl:function>
  
  <xsl:function name="Q{f}text" as="text()">
    <xsl:param name="value" as="xs:string"/>
    <xsl:value-of select="$value"/>
  </xsl:function>
  
  <!-- "treat as" applied to constructed attribute nodes -->
  
  <xsl:template name="c-107" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(outermost(//PRICE) ! Q{f}attribute('x', string(.))) treat as attribute(x)+"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <xsl:template name="c-107a" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(outermost(//PRICE) ! Q{f}attribute('x', string(.))) treat as element(x)+"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- "treat as" applied to constructed element nodes -->
  
  <xsl:template name="c-108" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(outermost(//PRICE) ! Q{f}element('x', string(.))) treat as element(x)+"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <xsl:template name="c-108a" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(outermost(//PRICE) ! Q{f}element('x', string(.))) treat as attribute(x)+"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <!-- "treat as" applied to constructed text nodes -->
  
  <xsl:template name="c-109" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(outermost(//PRICE) ! Q{f}text(string(.)||'$ ')) treat as text()+"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  
  <xsl:template name="c-109a" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <out>
        <xsl:value-of select="(outermost(//PRICE) ! Q{f}text(string(.))) treat as comment()+"/>
      </out>
    </xsl:source-document>
  </xsl:template>
  

</xsl:stylesheet>
