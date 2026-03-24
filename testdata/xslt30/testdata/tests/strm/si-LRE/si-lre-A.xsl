<xsl:transform version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="map xs">

  <xsl:variable name="RUN" select="true()" static="yes"/>
  <xsl:strip-space elements="*"/>

  <!-- within xsl:source-document, use LRE: atomic values, consuming -->

  <xsl:template name="cy-001" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:for-each select="account/transaction[@value &lt; 0]/@value">
          <value>
            <xsl:sequence select="data(.)"/>
          </value>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use LRE: atomic values, consuming and non-consuming -->

  <xsl:template name="cy-002" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:for-each select="data(account/transaction[@value &lt; 0]/@value), 101, 102">
          <e>
            <xsl:value-of select="."/>
          </e>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use LRE: climbing posture -->

  <xsl:template name="cy-003" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:variable name="atts" as="element(*)*">
          <xsl:for-each select="account/transaction[@value &lt; 0]/@value">
            <transaction>
              <xsl:value-of select="."/>
            </transaction>
          </xsl:for-each>
        </xsl:variable>
        <xsl:copy-of select="data($atts)"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use LRE: climbing posture -->

  <xsl:template name="cy-004" use-when="$RUN">
    <xsl:variable name="extra" as="element()*">
      <PRICE value="101"/>
      <PRICE value="102"/>
    </xsl:variable>
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:variable name="atts" as="element(*)*">
          <xsl:for-each select="account/transaction[@value &lt; 0]/@value, $extra/@value">
            <transaction>
              <xsl:value-of select="."/>
            </transaction>
          </xsl:for-each>
        </xsl:variable>
        <xsl:copy-of select="data($atts)"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use LRE: striding posture, element nodes -->

  <xsl:template name="cy-005" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="/BOOKLIST/BOOKS/ITEM/PRICE">
          <PRICE>
            <xsl:value-of select="."/>
          </PRICE>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use LRE: striding posture, text nodes -->

  <xsl:template name="cy-006" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="/BOOKLIST/BOOKS/ITEM/PRICE/text()">
          <PRICE>
            <xsl:value-of select="."/>
          </PRICE>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use LRE: striding posture, text nodes mixed with atomic values -->

  <xsl:template name="cy-007" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="/BOOKLIST/BOOKS/ITEM/PRICE/text(), 101, 102">
          <t>
            <xsl:value-of select="."/>
          </t>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use LRE: striding posture, element nodes mixed with grounded elements -->

  <xsl:template name="cy-008" use-when="$RUN">
    <xsl:variable name="extra" as="element()*">
      <PRICE>100.00</PRICE>
      <PRICE>101.00</PRICE>
    </xsl:variable>
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="$extra, /BOOKLIST/BOOKS/ITEM/PRICE">
          <t>
            <xsl:value-of select="."/>
          </t>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use LRE: descendant text nodes -->

  <xsl:template name="cy-009" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="outermost(//PRICE)">
          <PRICE>
            <xsl:sequence select="text()"/>
          </PRICE>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use LRE: descendant text nodes mixed with atomic values -->

  <xsl:template name="cy-010" use-when="true() or $RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="100, 101, /BOOKLIST/BOOKS/ITEM/PRICE">
          <t>
            <xsl:value-of select="if (. instance of element()) then text() else ."/>
          </t>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use LRE: whole document unchanged -->

  <xsl:template name="cy-011" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <head/>
        <doc>
          <xsl:copy-of select="child::node()"/>
        </doc>
        <tail/>
      </xsl:source-document>
    </out>
  </xsl:template>



  <!-- within xsl:source-document, use LRE: validation="strip" (non-schema-aware) -->

  <xsl:template name="cy-022" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:for-each select="/*/*:description">
          <description xsl:validation="strip">
            <xsl:value-of select="."/>
          </description>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use LRE: validation="preserve" (non-schema-aware) -->

  <xsl:template name="cy-023" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:for-each select="/*/*:description">
          <description xsl:validation="preserve">
            <xsl:value-of select="."/>
          </description>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use LRE: validation="lax" (non-schema-aware) -->

  <xsl:template name="cy-024" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:for-each select="/*/*:description">
          <description xsl:validation="lax">
            <xsl:value-of select="."/>
          </description>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use LRE: copy-namespaces="yes", argument grounded and consuming -->

  <xsl:template name="cy-025" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:for-each select="copy-of(/*/*:description)">
          <x:description xmlns:x="http://cy-025.com/">
            <xsl:value-of select="."/>
          </x:description>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use LRE: copy-namespaces="no", argument grounded and consuming -->

  <xsl:template name="cy-026" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:for-each select="copy-of(/*/*:description)">
          <gml:description xmlns:gml="http://cy-026.com/">
            <xsl:value-of select="."/>
          </gml:description>
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
          <dddd xsl:use-attribute-sets="as-1">
            <xsl:sequence select="*"/>
          </dddd>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use LRE/on-empty: empty sequence selected -->

  <xsl:template name="cy-040" use-when="$RUN">
    <out>
      <xsl:variable name="a" as="element()">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:where-populated>
          <a>
            <xsl:sequence select="/*[@dummy='not-there']"/>
          </a>
        </xsl:where-populated>
        <xsl:on-empty select="$a"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use LRE/on-empty: empty element constructed -->

  <xsl:template name="cy-041" use-when="$RUN">
    <out>
      <xsl:variable name="a" as="element()">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:where-populated>
          <a>
            <xsl:if test="current-date() lt xs:date('1900-01-01')">
              <b/>
            </xsl:if>
          </a>
        </xsl:where-populated>
        <xsl:on-empty select="$a"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use LRE/on-empty: empty document node constructed -->

  <xsl:template name="cy-042" use-when="$RUN">
    <out>
      <xsl:variable name="a" as="element()">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:where-populated>
          <a>
            <xsl:if test="current-date() lt xs:date('1900-01-01')">
              <b/>
            </xsl:if>
          </a>
        </xsl:where-populated>
        <xsl:on-empty select="$a"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use LRE/on-empty: empty document node constructed -->

  <xsl:template name="cy-043" use-when="$RUN">
    <out>
      <xsl:variable name="a" as="element()">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:where-populated>
          <a>
            <xsl:if test="current-date() lt xs:date('1900-01-01')">
              <b/>
            </xsl:if>
          </a>
        </xsl:where-populated>
        <xsl:on-empty select="$a"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use LRE/on-empty: empty comment node constructed -->

  <xsl:template name="cy-044" use-when="$RUN">
    <out>
      <xsl:variable name="a" as="element()">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/special.xml">
        <xsl:where-populated>
          <a>
            <xsl:if test="current-date() lt xs:date('1900-01-01')">
              <xsl:sequence select="special/comment()[2]"/>
            </xsl:if>
          </a>
        </xsl:where-populated>
        <xsl:on-empty select="$a"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use LRE/on-empty: empty PI node constructed -->

  <xsl:template name="cy-045" use-when="$RUN">
    <out>
      <xsl:variable name="a" as="element()">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/special.xml">
        <xsl:where-populated>
          <a>
            <xsl:if test="current-date() lt xs:date('1900-01-01')">
              <xsl:sequence select="special/processing-instruction()[2]"/>
            </xsl:if>
          </a>
        </xsl:where-populated>
        <xsl:on-empty select="$a"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use LRE/on-empty: empty element node constructed -->

  <xsl:template name="cy-046" use-when="$RUN">
    <out>
      <xsl:variable name="a" as="element()">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/special.xml">
        <xsl:for-each select="special">
          <xsl:where-populated>
            <special>
              <xsl:sequence select="f/@b"/>
            </special>
          </xsl:where-populated>
          <xsl:on-empty select="$a"/>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use LRE/on-empty: empty element constructed -->

  <xsl:template name="cy-047" use-when="$RUN">
    <out>
      <xsl:variable name="a" as="element()">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:for-each select="*">
          <xsl:where-populated>
            <dddd>
              <xsl:copy-of select="a/b/c/d/e/f/g"/>
            </dddd>
          </xsl:where-populated>
          <xsl:on-empty select="$a"/>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use LRE/on-empty: empty document constructed -->

  <xsl:template name="cy-048" use-when="$RUN">
    <out>
      <xsl:variable name="a" as="element()">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:for-each select="*">
          <xsl:where-populated>
            <dddd>
              <xsl:copy-of select="a/b/c/d/e/f/g"/>
            </dddd>
          </xsl:where-populated>
          <xsl:on-empty select="$a"/>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>


</xsl:transform>
