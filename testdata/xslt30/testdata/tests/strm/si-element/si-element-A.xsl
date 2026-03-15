<xsl:transform version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="map xs">

  <xsl:variable name="RUN" select="true()" static="yes"/>
  <xsl:strip-space elements="*"/>

  <!-- within xsl:source-document, use xsl:element: atomic values, consuming -->

  <xsl:template name="cy-001" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:for-each select="account/transaction[@value &lt; 0]/@value">
          <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}">
            <xsl:sequence select="data(.)"/>
          </xsl:element>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: atomic values, consuming and non-consuming -->

  <xsl:template name="cy-002" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:for-each select="data(account/transaction[@value &lt; 0]/@value), 101, 102">
          <xsl:element name="e">
            <xsl:value-of select="."/>
          </xsl:element>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: climbing posture -->

  <xsl:template name="cy-003" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:variable name="atts" as="element(*)*">
          <xsl:for-each select="account/transaction[@value &lt; 0]/@value">
            <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}">
              <xsl:value-of select="."/>
            </xsl:element>
          </xsl:for-each>
        </xsl:variable>
        <xsl:copy-of select="data($atts)"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: climbing posture -->

  <xsl:template name="cy-004" use-when="$RUN">
    <xsl:variable name="extra" as="element()*">
      <PRICE value="101"/>
      <PRICE value="102"/>
    </xsl:variable>
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:variable name="atts" as="element(*)*">
          <xsl:for-each select="account/transaction[@value &lt; 0]/@value, $extra/@value">
            <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}">
              <xsl:value-of select="."/>
            </xsl:element>
          </xsl:for-each>
        </xsl:variable>
        <xsl:copy-of select="data($atts)"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: striding posture, element nodes -->

  <xsl:template name="cy-005" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="/BOOKLIST/BOOKS/ITEM/PRICE">
          <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}">
            <xsl:value-of select="."/>
          </xsl:element>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: striding posture, text nodes -->

  <xsl:template name="cy-006" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="/BOOKLIST/BOOKS/ITEM/PRICE/text()">
          <xsl:element name="{name(..)}" namespace="{namespace-uri(..)}">
            <xsl:value-of select="."/>
          </xsl:element>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: striding posture, text nodes mixed with atomic values -->

  <xsl:template name="cy-007" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="/BOOKLIST/BOOKS/ITEM/PRICE/text(), 101, 102">
          <xsl:element name="t">
            <xsl:value-of select="."/>
          </xsl:element>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: striding posture, element nodes mixed with grounded elements -->

  <xsl:template name="cy-008" use-when="$RUN">
    <xsl:variable name="extra" as="element()*">
      <PRICE>100.00</PRICE>
      <PRICE>101.00</PRICE>
    </xsl:variable>
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="$extra, /BOOKLIST/BOOKS/ITEM/PRICE">
          <xsl:element name="t">
            <xsl:value-of select="."/>
          </xsl:element>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: descendant text nodes -->

  <xsl:template name="cy-009" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="outermost(//PRICE)">
          <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}">
            <xsl:sequence select="text()"/>
          </xsl:element>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: descendant text nodes mixed with atomic values -->

  <xsl:template name="cy-010" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="100, 101, /BOOKLIST/BOOKS/ITEM/PRICE">
          <xsl:element name="t">
            <xsl:value-of select="if (. instance of element()) then text() else ."/>
          </xsl:element>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: whole document unchanged -->

  <xsl:template name="cy-011" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <head/>
        <xsl:element name="doc">
          <xsl:copy-of select="child::node()"/>
        </xsl:element>
        <tail/>
      </xsl:source-document>
    </out>
  </xsl:template>



  <!-- within xsl:source-document, use xsl:element: validation="strip" (non-schema-aware) -->

  <xsl:template name="cy-022" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:for-each select="/*/*:description">
          <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}" validation="strip">
            <xsl:value-of select="."/>
          </xsl:element>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: validation="preserve" (non-schema-aware) -->

  <xsl:template name="cy-023" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:for-each select="/*/*:description">
          <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}" validation="preserve">
            <xsl:value-of select="."/>
          </xsl:element>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: validation="lax" (non-schema-aware) -->

  <xsl:template name="cy-024" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:for-each select="/*/*:description">
          <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}" validation="lax">
            <xsl:value-of select="."/>
          </xsl:element>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: copy-namespaces="yes", argument grounded and consuming -->

  <xsl:template name="cy-025" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:for-each select="copy-of(/*/*:description)">
          <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}">
            <xsl:value-of select="."/>
          </xsl:element>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: copy-namespaces="no", argument grounded and consuming -->

  <xsl:template name="cy-026" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:for-each select="copy-of(/*/*:description)">
          <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}">
            <xsl:value-of select="."/>
          </xsl:element>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>


  <!-- xsl:element referring to a streamable attribute set -->

  <xsl:attribute-set name="as-1" streamable="yes">
    <xsl:attribute name="x" select="1"/>
    <xsl:attribute name="y" select="2"/>
  </xsl:attribute-set>

  <xsl:template name="cy-029" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:for-each select="*">
          <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}" use-attribute-sets="as-1">
            <xsl:sequence select="*"/>
          </xsl:element>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element/on-empty: empty sequence selected -->

  <xsl:template name="cy-040" use-when="$RUN">
    <out>
      <xsl:variable name="a" as="element()">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:where-populated>
          <xsl:element name="{name($a)}">
            <xsl:sequence select="/*[@dummy='not-there']"/>
          </xsl:element>
        </xsl:where-populated>
        <xsl:on-empty select="$a"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element/on-empty: empty element constructed -->

  <xsl:template name="cy-041" use-when="$RUN">
    <out>
      <xsl:variable name="a" as="element()">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:where-populated>
          <xsl:element name="{name($a)}">
            <xsl:if test="current-date() lt xs:date('1900-01-01')">
              <b/>
            </xsl:if>
          </xsl:element>
        </xsl:where-populated>
        <xsl:on-empty select="$a"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element/on-empty: empty document node constructed -->

  <xsl:template name="cy-042" use-when="$RUN">
    <out>
      <xsl:variable name="a" as="element()">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:where-populated>
          <xsl:element name="{name($a)}">
            <xsl:if test="current-date() lt xs:date('1900-01-01')">
              <b/>
            </xsl:if>
          </xsl:element>
        </xsl:where-populated>
        <xsl:on-empty select="$a"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element/on-empty: empty document node constructed -->

  <xsl:template name="cy-043" use-when="$RUN">
    <out>
      <xsl:variable name="a" as="element()">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:where-populated>
          <xsl:element name="{name($a)}">
            <xsl:if test="current-date() lt xs:date('1900-01-01')">
              <b/>
            </xsl:if>
          </xsl:element>
        </xsl:where-populated>
        <xsl:on-empty select="$a"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element/on-empty: empty comment node constructed -->

  <xsl:template name="cy-044" use-when="$RUN">
    <out>
      <xsl:variable name="a" as="element()">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/special.xml">
        <xsl:where-populated>
          <xsl:element name="{name($a)}" namespace="{namespace-uri($a)}">
            <xsl:if test="current-date() lt xs:date('1900-01-01')">
              <xsl:sequence select="special/comment()[2]"/>
            </xsl:if>
          </xsl:element>
        </xsl:where-populated>
        <xsl:on-empty select="$a"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element/on-empty: empty PI node constructed -->

  <xsl:template name="cy-045" use-when="$RUN">
    <out>
      <xsl:variable name="a" as="element()">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/special.xml">
        <xsl:where-populated>
          <xsl:element name="{name($a)}" namespace="{namespace-uri($a)}">
            <xsl:if test="current-date() lt xs:date('1900-01-01')">
              <xsl:sequence select="special/processing-instruction()[2]"/>
            </xsl:if>
          </xsl:element>
        </xsl:where-populated>
        <xsl:on-empty select="$a"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element/on-empty: empty element node constructed -->

  <xsl:template name="cy-046" use-when="$RUN">
    <out>
      <xsl:variable name="a" as="element()">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/special.xml">
        <xsl:where-populated>
          <xsl:for-each select="special">
            <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}">
              <xsl:sequence select="f/@b"/>
            </xsl:element>
          </xsl:for-each>
        </xsl:where-populated>
        <xsl:on-empty select="$a"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element/on-empty: empty element constructed -->

  <xsl:template name="cy-047" use-when="$RUN">
    <out>
      <xsl:variable name="a" as="element()">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:where-populated>
          <xsl:for-each select="*">
            <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}">
              <xsl:copy-of select="a/b/c/d/e/f/g"/>
            </xsl:element>
          </xsl:for-each>
        </xsl:where-populated>
        <xsl:on-empty select="$a"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element/on-empty: empty document constructed -->

  <xsl:template name="cy-048" use-when="$RUN">
    <out>
      <xsl:variable name="a" as="element()">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:where-populated>
          <xsl:for-each select="*">
            <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}">
              <xsl:copy-of select="a/b/c/d/e/f/g"/>
            </xsl:element>
          </xsl:for-each>
        </xsl:where-populated>
        <xsl:on-empty select="$a"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:element (XSLT equivalent of test case in Saxon bug 2550 -->
  
  <xsl:template name="cy-049" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:for-each select="*/*">
          <xsl:element name="{name()}" namespace="{namespace-uri()}">
            <xsl:copy-of select="node()"/>
          </xsl:element>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>



</xsl:transform>
