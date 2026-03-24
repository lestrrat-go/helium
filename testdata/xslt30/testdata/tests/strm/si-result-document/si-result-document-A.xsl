<xsl:transform version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="map xs">

  <xsl:variable name="RUN" select="true()" static="yes"/>
  <xsl:strip-space elements="*"/>

  <!-- within xsl:source-document, use xsl:document: atomic values, consuming -->

  <xsl:template name="d-001" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/transactions.xml">
      <xsl:result-document href="d-001.xml">
        <out>
          <xsl:for-each select="account/transaction[@value &lt; 0]/@value">
            <xsl:sequence select="data(.)"/>
          </xsl:for-each>
        </out>
      </xsl:result-document>
    </xsl:source-document>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:document: atomic values, consuming and non-consuming -->

  <xsl:template name="d-002" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/transactions.xml">
      <xsl:result-document href="d-002.xml">
        <out>
          <xsl:for-each select="data(account/transaction[@value &lt; 0]/@value), 101, 102">
            <xsl:sequence select="data(.)"/>
          </xsl:for-each>
        </out>
      </xsl:result-document>
    </xsl:source-document>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:document: climbing posture -->

  <xsl:template name="d-003" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/transactions.xml">
      <xsl:result-document href="d-003.xml">
        <out>
          <xsl:for-each select="account/transaction[@value &lt; 0]/@value">
            <xsl:sequence select="data(.)"/>
          </xsl:for-each>
        </out>
      </xsl:result-document>
    </xsl:source-document>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:document: climbing posture -->

  <xsl:template name="d-004" use-when="$RUN">
    <xsl:variable name="extra" as="element()*">
      <PRICE value="101"/>
      <PRICE value="102"/>
    </xsl:variable>

    <xsl:source-document streamable="yes" href="../docs/transactions.xml">
      <xsl:result-document href="d-004.xml">
        <xsl:variable name="docs" as="item()*">
          <xsl:for-each select="account/transaction[@value &lt; 0]/@value, $extra/@value">
            <xsl:sequence select="data(.)"/>
          </xsl:for-each>
        </xsl:variable>
        <out>
          <xsl:copy-of select="data($docs)"/>
        </out>
      </xsl:result-document>
    </xsl:source-document>

  </xsl:template>

  <!-- within xsl:source-document, use xsl:document: striding posture, element nodes -->

  <xsl:template name="d-005" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <xsl:result-document href="d-005.xml">
        <out>
          <xsl:for-each select="/BOOKLIST/BOOKS/ITEM/PRICE">
            <xsl:copy-of select="."/>
          </xsl:for-each>
        </out>
      </xsl:result-document>
    </xsl:source-document>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:document: striding posture, text nodes -->

  <xsl:template name="d-006" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <xsl:result-document href="d-006.xml">
        <out>
          <xsl:for-each select="/BOOKLIST/BOOKS/ITEM/PRICE/text()">
            <xsl:copy-of select="., ','"/>
          </xsl:for-each>
        </out>
      </xsl:result-document>
    </xsl:source-document>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:document: striding posture, text nodes mixed with atomic values -->

  <xsl:template name="d-007" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <xsl:result-document href="d-007.xml">
        <out>
          <xsl:copy-of select="/BOOKLIST/BOOKS/ITEM/PRICE/text(), 101, 102"/>
        </out>
      </xsl:result-document>
    </xsl:source-document>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:document: striding posture, element nodes mixed with grounded elements -->

  <xsl:template name="d-008" use-when="$RUN">
    <xsl:variable name="extra" as="element()*">
      <PRICE>100.00</PRICE>
      <PRICE>101.00</PRICE>
    </xsl:variable>

    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <xsl:for-each select="$extra, /BOOKLIST/BOOKS/ITEM/PRICE">
        <xsl:result-document href="d-008-{position()}.xml">
          <xsl:copy-of select="."/>
        </xsl:result-document>
      </xsl:for-each>
    </xsl:source-document>

  </xsl:template>

  <!-- within xsl:source-document, use xsl:document: descendant text nodes -->

  <xsl:template name="d-009" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <xsl:for-each select="outermost(//PRICE)">
        <xsl:result-document href="d-009-{position()+2}.xml">
          <PRICE>
            <xsl:sequence select="text()"/>
          </PRICE>
        </xsl:result-document>
      </xsl:for-each>
    </xsl:source-document>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:document: descendant text nodes mixed with atomic values -->

  <xsl:template name="d-010" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <xsl:result-document href="d-010.xml">
        <out>
          <xsl:for-each select="100, 101, /BOOKLIST/BOOKS/ITEM/PRICE/text()">
            <xsl:sequence select="."/>
          </xsl:for-each>
        </out>
      </xsl:result-document>
    </xsl:source-document>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:document: whole document unchanged -->

  <xsl:template name="d-011" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
      <xsl:result-document href="d-011.xml">
        <out>
          <head/>
          <xsl:copy-of select="child::node()"/>
          <tail/>
        </out>
      </xsl:result-document>
    </xsl:source-document>
  </xsl:template>


  <!-- within xsl:source-document, use xsl:document: validation="strip" (non-schema-aware) -->

  <xsl:template name="d-022" use-when="$RUN">
    <xsl:source-document streamable="yes" href="../docs/citygml.xml">
      <xsl:result-document href="d-022.xml" validation="strip">
        <out>
          <xsl:for-each select="/*/*:description">
            <xsl:copy-of select="."/>
          </xsl:for-each>
        </out>
      </xsl:result-document>
    </xsl:source-document>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:document: validation="preserve" (non-schema-aware) -->

  <xsl:template name="d-023" use-when="$RUN">

    <xsl:source-document streamable="yes" href="../docs/citygml.xml">
      <xsl:result-document href="d-023.xml" validation="preserve">
        <out>
          <xsl:for-each select="/*/*:description">
            <xsl:copy-of select="."/>
          </xsl:for-each>
        </out>
      </xsl:result-document>
    </xsl:source-document>

  </xsl:template>

  <!-- within xsl:source-document, use xsl:document: validation="lax" (non-schema-aware) -->

  <xsl:template name="d-024" use-when="$RUN">

    <xsl:source-document streamable="yes" href="../docs/citygml.xml">
      <xsl:result-document href="d-024.xml" validation="lax">
        <out>
          <xsl:for-each select="/*/*:description">
            <xsl:copy-of select="."/>
          </xsl:for-each>
        </out>
      </xsl:result-document>
    </xsl:source-document>

  </xsl:template>



  <!-- within xsl:source-document, use xsl:document/on-empty: empty sequence selected ->
  
  <xsl:template name="d-040" use-when="$RUN">
    <out>
      <xsl:variable name="a"><a/></xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:result-document href="d-040.xml" select="/*[@dummy='not-there']" on-empty="$a">
          <b/>
        </xsl:result-document>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!- within xsl:source-document, use xsl:document/on-empty: empty element constructed ->
  
  <xsl:template name="d-041" use-when="$RUN">
    <out>
      <xsl:variable name="a"><a/></xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:result-document href="d-041.xml" select="/*" on-empty="$a">
          <xsl:if test="current-date() lt xs:date('1900-01-01')"><b/></xsl:if>
        </xsl:result-document>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!- within xsl:source-document, use xsl:document/on-empty: empty document node constructed ->
  
  <xsl:template name="d-042" use-when="$RUN">
    <out>
      <xsl:variable name="a"><a/></xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:result-document href="d-042.xml" on-empty="$a">
          <xsl:if test="current-date() lt xs:date('1900-01-01')"><b/></xsl:if>
        </xsl:result-document>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!- within xsl:source-document, use xsl:document/on-empty: empty document node constructed ->
  
  <xsl:template name="d-043" use-when="$RUN">
    <out>
      <xsl:variable name="a"><a/></xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:result-document href="d-043.xml" on-empty="$a">
          <xsl:if test="current-date() lt xs:date('1900-01-01')"><b/></xsl:if>
        </xsl:result-document>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!- within xsl:source-document, use xsl:document/on-empty: empty comment node constructed ->
  
  <xsl:template name="d-044" use-when="$RUN">
    <out>
      <xsl:variable name="a"><a/></xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/special.xml">
        <xsl:result-document href="d-044.xml" select="special/comment()[2]" on-empty="$a">
          <xsl:if test="current-date() lt xs:date('1900-01-01')"><b/></xsl:if>
        </xsl:result-document>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!- within xsl:source-document, use xsl:document/on-empty: empty PI node constructed ->
  
  <xsl:template name="d-045" use-when="$RUN">
    <out>
      <xsl:variable name="a"><a/></xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/special.xml">
        <xsl:result-document href="d-045.xml" select="special/processing-instruction()[2]" on-empty="$a">
          <xsl:if test="current-date() lt xs:date('1900-01-01')"><b/></xsl:if>
        </xsl:result-document>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!- within xsl:source-document, use xsl:document/on-empty: empty attribute node constructed ->
  
  <xsl:template name="d-046" use-when="$RUN">
    <out>
      <xsl:variable name="a"><a/></xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/special.xml">
        <xsl:result-document href="d-046.xml" select="special/f/@a" on-empty="$a"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!- within xsl:source-document, use xsl:document/on-empty: empty element constructed ->
  
  <xsl:template name="d-047" use-when="$RUN">
    <out>
      <xsl:variable name="a"><a/></xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:result-document href="d-047.xml" select="/*" on-empty="$a">
          <xsl:copy-of select="a/b/c/d/e/f/g"/>
        </xsl:result-document>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!- within xsl:source-document, use xsl:document/on-empty: empty document constructed ->
  
  <xsl:template name="d-048" use-when="$RUN">
    <out>
      <xsl:variable name="a"><a/></xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:result-document href="d-048.xml" on-empty="$a">
          <xsl:copy-of select="a/b/c/d/e/f/g"/>
        </xsl:result-document>
      </xsl:source-document>
    </out>
  </xsl:template>
  
   -->

</xsl:transform>
