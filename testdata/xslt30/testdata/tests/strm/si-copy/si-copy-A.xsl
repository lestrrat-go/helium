<xsl:transform version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="map xs">

  <xsl:variable name="RUN" select="true()" static="yes"/>
  <xsl:strip-space elements="*"/>

  <!-- within xsl:source-document, use xsl:copy: atomic values, consuming -->

  <xsl:template name="cy-001" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:for-each select="account/transaction[@value &lt; 0]/@value">
          <xsl:copy select="data(.)"/>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy: atomic values, consuming and non-consuming -->

  <xsl:template name="cy-002" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:for-each select="data(account/transaction[@value &lt; 0]/@value), 101, 102">
          <xsl:copy/>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy: climbing posture -->

  <xsl:template name="cy-003" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:variable name="atts" as="attribute(*)*">
          <xsl:for-each select="account/transaction[@value &lt; 0]/@value">
            <xsl:copy/>
          </xsl:for-each>
        </xsl:variable>
        <xsl:copy-of select="data($atts)"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy: climbing posture -->

  <xsl:template name="cy-004" use-when="$RUN">
    <xsl:variable name="extra" as="element()*">
      <PRICE value="101"/>
      <PRICE value="102"/>
    </xsl:variable>
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:variable name="atts" as="attribute(*)*">
          <xsl:for-each select="account/transaction[@value &lt; 0]/@value, $extra/@value">
            <xsl:copy/>
          </xsl:for-each>
        </xsl:variable>
        <xsl:copy-of select="data($atts)"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy: striding posture, element nodes -->

  <xsl:template name="cy-005" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="/BOOKLIST/BOOKS/ITEM/PRICE">
          <xsl:copy>
            <xsl:value-of select="."/>
          </xsl:copy>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy: striding posture, text nodes -->

  <xsl:template name="cy-006" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="/BOOKLIST/BOOKS/ITEM/PRICE/text()">
          <xsl:copy/>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy: striding posture, text nodes mixed with atomic values -->

  <xsl:template name="cy-007" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="/BOOKLIST/BOOKS/ITEM/PRICE/text(), 101, 102">
          <xsl:copy/>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy: striding posture, element nodes mixed with grounded elements -->

  <xsl:template name="cy-008" use-when="$RUN">
    <xsl:variable name="extra" as="element()*">
      <PRICE>100.00</PRICE>
      <PRICE>101.00</PRICE>
    </xsl:variable>
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="$extra, /BOOKLIST/BOOKS/ITEM/PRICE">
          <xsl:copy>
            <xsl:value-of select="."/>
          </xsl:copy>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy: descendant text nodes -->

  <xsl:template name="cy-009" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="outermost(//PRICE)">
          <xsl:copy select="text()"/>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy: descendant text nodes mixed with atomic values -->

  <xsl:template name="cy-010" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="100, 101, /BOOKLIST/BOOKS/ITEM/PRICE">
          <xsl:copy>
            <xsl:value-of select="text()"/>
          </xsl:copy>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy: whole document unchanged -->

  <xsl:template name="cy-011" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <head/>
        <xsl:copy>
          <xsl:copy-of select="child::node()"/>
        </xsl:copy>
        <tail/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy: copy-namespaces=no -->

  <xsl:template name="cy-020" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:for-each select="/*/*:description">
          <xsl:copy copy-namespaces="no">
            <xsl:value-of select="."/>
          </xsl:copy>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy: copy-namespaces=yes -->

  <xsl:template name="cy-021" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:for-each select="/*/*:description">
          <xsl:copy copy-namespaces="yes">
            <xsl:value-of select="."/>
          </xsl:copy>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy: validation="strip" (non-schema-aware) -->

  <xsl:template name="cy-022" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:for-each select="/*/*:description">
          <xsl:copy copy-namespaces="no" validation="strip">
            <xsl:value-of select="."/>
          </xsl:copy>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy: validation="preserve" (non-schema-aware) -->

  <xsl:template name="cy-023" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:for-each select="/*/*:description">
          <xsl:copy copy-namespaces="no" validation="preserve">
            <xsl:value-of select="."/>
          </xsl:copy>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy: validation="lax" (non-schema-aware) -->

  <xsl:template name="cy-024" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:for-each select="/*/*:description">
          <xsl:copy copy-namespaces="no" validation="lax">
            <xsl:value-of select="."/>
          </xsl:copy>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy: copy-namespaces="yes", argument grounded and consuming -->

  <xsl:template name="cy-025" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:for-each select="copy-of(/*/*:description)">
          <xsl:copy copy-namespaces="yes">
            <xsl:value-of select="."/>
          </xsl:copy>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy: copy-namespaces="no", argument grounded and consuming -->

  <xsl:template name="cy-026" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:for-each select="copy-of(/*/*:description)">
          <xsl:copy copy-namespaces="no">
            <xsl:value-of select="."/>
          </xsl:copy>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy: type error, more than one item selected -->

  <xsl:template name="cy-027" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:copy select="/*/*"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy: nested -->

  <xsl:template name="cy-028" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:copy>
          <xsl:copy select="*">
            <xsl:copy select="*:cityObjectMember[1]">
              <xsl:copy select="*[1]">
                <xsl:copy select="*[1]"/>
              </xsl:copy>
            </xsl:copy>
          </xsl:copy>
        </xsl:copy>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- xsl:copy referring to a streamable attribute set -->

  <xsl:attribute-set name="as-1" streamable="yes">
    <xsl:attribute name="x" select="1"/>
    <xsl:attribute name="y" select="2"/>
  </xsl:attribute-set>

  <xsl:template name="cy-029" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:copy select="*" use-attribute-sets="as-1"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy/on-empty: empty sequence selected -->

  <xsl:template name="cy-040" use-when="$RUN">
    <out>
      <xsl:variable name="a">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:where-populated>
          <xsl:copy select="/*[@dummy='not-there']">
            <b/>
          </xsl:copy>
        </xsl:where-populated>
        <xsl:on-empty select="$a/a"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy/on-empty: empty element constructed -->

  <xsl:template name="cy-041" use-when="$RUN">
    <out>
      <xsl:variable name="a">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:where-populated>
          <xsl:copy select="/*">
            <xsl:if test="current-date() lt xs:date('1900-01-01')">
              <b/>
            </xsl:if>
          </xsl:copy>
        </xsl:where-populated>
        <xsl:on-empty select="$a/a"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy/on-empty: empty document node constructed -->

  <xsl:template name="cy-042" use-when="$RUN">
    <out>
      <xsl:variable name="a">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:where-populated>
          <xsl:copy>
            <xsl:if test="current-date() lt xs:date('1900-01-01')">
              <b/>
            </xsl:if>
          </xsl:copy>
        </xsl:where-populated>
        <xsl:on-empty select="$a/a"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy/on-empty: empty document node constructed -->

  <xsl:template name="cy-043" use-when="$RUN">
    <out>
      <xsl:variable name="a">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:where-populated>
          <xsl:copy>
            <xsl:if test="current-date() lt xs:date('1900-01-01')">
              <b/>
            </xsl:if>
          </xsl:copy>
        </xsl:where-populated>
        <xsl:on-empty select="$a/a"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy/on-empty: empty comment node constructed -->

  <xsl:template name="cy-044" use-when="$RUN">
    <out>
      <xsl:variable name="a">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/special.xml">
        <xsl:where-populated>
          <xsl:copy select="special/comment()[2]">
            <xsl:if test="current-date() lt xs:date('1900-01-01')">
              <b/>
            </xsl:if>
          </xsl:copy>
        </xsl:where-populated>
        <xsl:on-empty select="$a/a"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy/on-empty: empty PI node constructed -->

  <xsl:template name="cy-045" use-when="$RUN">
    <out>
      <xsl:variable name="a">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/special.xml">
        <xsl:where-populated>
          <xsl:copy select="special/processing-instruction()[2]">
            <xsl:if test="current-date() lt xs:date('1900-01-01')">
              <b/>
            </xsl:if>
          </xsl:copy>
        </xsl:where-populated>
        <xsl:on-empty select="$a/a"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy/on-empty: empty attribute node constructed -->

  <xsl:template name="cy-046" use-when="$RUN">
    <out>
      <xsl:variable name="a">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/special.xml">
        <xsl:copy select="special/f/@a"/>
        <xsl:on-empty select="$a/a"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy/on-empty: empty element constructed -->

  <xsl:template name="cy-047" use-when="$RUN">
    <out>
      <xsl:variable name="a">
        <is-empty/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:copy select="/*">
          <xsl:copy-of select="a/b/c/d/e/f/g"/>
          <xsl:on-empty select="$a/is-empty"/>
        </xsl:copy>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy/on-empty: empty document constructed -->

  <xsl:template name="cy-048" use-when="$RUN">
    <out>
      <xsl:variable name="a">
        <is-empty/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:copy>
          <xsl:copy-of select="a/b/c/d/e/f/g"/>
          <xsl:on-empty select="$a/is-empty"/>
        </xsl:copy>
      </xsl:source-document>
    </out>
  </xsl:template>
  
    <!-- within xsl:source-document, use xsl:copy with xsl:where-populated: empty element constructed -->

  <xsl:template name="cy-049" use-when="$RUN">
    <out>
      <xsl:variable name="a">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:where-populated>
          <xsl:copy select="/*">
            <xsl:copy-of select="a/b/c/d/e/f/g"/>
          </xsl:copy>
        </xsl:where-populated>  
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:copy with xsl:where-populated: empty document constructed -->

  <xsl:template name="cy-050" use-when="$RUN">
    <out>
      <xsl:variable name="a">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:where-populated>
          <xsl:copy>
            <xsl:copy-of select="a/b/c/d/e/f/g"/>
          </xsl:copy>
        </xsl:where-populated>  
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy/on-empty: empty PI node constructed (body of xsl:copy is ignored for a PI node) -->
  
  <xsl:template name="cy-051" use-when="$RUN">
    <out>
      <xsl:variable name="a">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/special.xml">
        <xsl:where-populated>
          <xsl:copy select="special/processing-instruction()[2]">
            <xsl:if test="current-date() gt xs:date('1900-01-01')">
              <b/>
            </xsl:if>
          </xsl:copy>
        </xsl:where-populated>
        <xsl:on-empty select="$a/a"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- Body of xsl:copy ignored for a processing instruction node -->
  
  <xsl:template name="cy-052" use-when="$RUN">
    <out>
      <xsl:variable name="a">
        <a/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/special.xml">
          <xsl:copy select="special/node()[4]">
            <xsl:copy-of select="child::node()"/>
          </xsl:copy>
      </xsl:source-document>
    </out>
  </xsl:template>




</xsl:transform>
