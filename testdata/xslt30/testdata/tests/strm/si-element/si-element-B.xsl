<xsl:transform version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  xmlns:err="http://www.w3.org/2005/xqt-errors" xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="map xs err">

  <xsl:variable name="RUN" select="true()" static="yes"/>
  <xsl:strip-space elements="*"/>

  <xsl:import-schema schema-location="../docs/loans.xsd"/>

  <!-- within xsl:source-document, use xsl:element: schema-aware, validation=strict -->

  <xsl:template name="cy-101" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:for-each select="*">
          <xsl:variable name="copy">
            <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}" validation="strict">
              <xsl:copy-of select="child::node()"/>
            </xsl:element>
          </xsl:variable>
          <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: schema-aware, validation=lax -->

  <xsl:template name="cy-102" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:for-each select="*">
          <xsl:variable name="copy">
            <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}" validation="lax">
              <xsl:copy-of select="child::node()"/>
            </xsl:element>
          </xsl:variable>
          <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: schema-aware, validation=strip -->

  <xsl:template name="cy-103" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:for-each select="*">
          <xsl:variable name="copy">
            <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}" validation="strip">
              <xsl:copy-of select="child::node()"/>
            </xsl:element>
          </xsl:variable>
          <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: schema-aware, validation=preserve -->

  <xsl:template name="cy-104" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:for-each select="*">
          <xsl:variable name="copy">
            <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}" validation="preserve">
              <xsl:copy-of select="child::node()"/>
            </xsl:element>
          </xsl:variable>
          <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: schema-aware, validation by type -->

  <xsl:template name="cy-105" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:for-each select="*">
          <xsl:variable name="copy">
            <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}" type="myroot">
              <xsl:copy-of select="child::node()"/>
            </xsl:element>
          </xsl:variable>
          <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: schema-aware, validation=strict, no element declaration available -->

  <xsl:template name="cy-106" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:for-each select="*">
          <xsl:variable name="copy">
            <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}" validation="strict">
              <xsl:copy-of select="child::node()"/>
            </xsl:element>
          </xsl:variable>
          <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: schema-aware, validation=lax, no element declaration available -->

  <xsl:template name="cy-107" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:for-each select="*">
          <xsl:variable name="copy">
            <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}" validation="lax">
              <xsl:copy-of select="child::node()"/>
            </xsl:element>
          </xsl:variable>
          <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: schema-aware, validation by type, invalid (wrong namespace) -->

  <xsl:template name="cy-108" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:for-each select="*">
          <xsl:variable name="copy">
            <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}" type="myroot">
              <xsl:copy-of select="child::node()"/>
            </xsl:element>
          </xsl:variable>
          <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: schema-aware, validation=strict, no element declaration available, caught -->

  <xsl:template name="cy-109" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:for-each select="*">
          <xsl:try>
            <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}" validation="strict">
              <xsl:copy-of select="child::node()"/>
            </xsl:element>
            <xsl:catch errors="*">
              <xsl:value-of select="local-name-from-QName($err:code)"/>
            </xsl:catch>
          </xsl:try>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: schema-aware, validation by type, invalid, caught -->

  <xsl:template name="cy-110" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:for-each select="*">
          <xsl:try>
            <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}" type="myroot">
              <xsl:copy-of select="child::node()"/>
            </xsl:element>
            <xsl:catch errors="*">
              <xsl:value-of select="local-name-from-QName($err:code)"/>
            </xsl:catch>
          </xsl:try>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: schema-aware, validation=strict, argument is consuming and grounded -->

  <xsl:template name="cy-111" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:for-each select="copy-of(/*/*)">
            <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}" validation="strict">
              <xsl:copy-of select="child::node()"/>
            </xsl:element>
          </xsl:for-each>
        </xsl:variable>
        <xsl:value-of select="$copy instance of element(*, ACERequest)*"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: schema-aware, validation=lax, argument is consuming and grounded -->

  <xsl:template name="cy-112" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:for-each select="copy-of(/*/*)">
            <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}" validation="lax">
              <xsl:copy-of select="child::node()"/>
            </xsl:element>
          </xsl:for-each>
        </xsl:variable>
        <xsl:value-of select="$copy instance of element(*, ACERequest)"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: schema-aware, validation=strip, argument is consuming and grounded -->

  <xsl:template name="cy-113" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:for-each select="copy-of(/*/*)">
            <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}" validation="strip">
              <xsl:copy-of select="child::node()"/>
            </xsl:element>
          </xsl:for-each>
        </xsl:variable>
        <xsl:value-of select="$copy instance of element(*, ACERequest)"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: schema-aware, validation=preserve, argument is consuming and grounded -->

  <xsl:template name="cy-114" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:for-each select="copy-of(/*/*)">
            <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}" validation="preserve">
              <xsl:copy-of select="child::node()"/>
            </xsl:element>
          </xsl:for-each>
        </xsl:variable>
        <xsl:value-of select="$copy instance of element(*, ACERequest)"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: schema-aware, validation by type, argument is consuming and grounded -->

  <xsl:template name="cy-115" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:for-each select="copy-of(/*/*)">
            <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}" type="ACERequest">
              <xsl:copy-of select="child::node()"/>
            </xsl:element>
          </xsl:for-each>
        </xsl:variable>
        <xsl:value-of select="$copy instance of element(*, ACERequest)"/>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: schema-aware, validate attribute node by type, success -->

  <xsl:template name="cy-116" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:for-each select="*">
          <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}" type="xs:decimal">
            <xsl:value-of select="*/@version"/>
          </xsl:element>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: schema-aware, validate attribute node by type, failure -->

  <xsl:template name="cy-117" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:for-each select="*">
          <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}" type="xs:date">
            <xsl:value-of select="*/@version"/>
          </xsl:element>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: schema-aware, validation=strict, on-empty must be empty or valid -->

  <xsl:template name="cy-118" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
          <xsl:where-populated>
            <xsl:for-each select="head(//PoolIdentifier)">
              <xsl:element name="processingDate" validation="strict">
                <xsl:copy-of select="*/*/NONEXISTENT"/>
                <xsl:on-empty select="'2015-08-11'"/>
              </xsl:element>
            </xsl:for-each>
          </xsl:where-populated>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: schema-aware, validation=strict, on-empty must be empty or valid -->

  <xsl:template name="cy-119" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:where-populated>
            <xsl:for-each select="head(//PoolIdentifier)">
              <xsl:element  name="processingDate" validation="strict">
                <xsl:copy-of select="*/*/NONEXISTENT"/>
                <xsl:on-empty select="'NOT_A_NUMBER'"/>
              </xsl:element>
            </xsl:for-each>
          </xsl:where-populated>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: schema-aware, validation by type, on-empty must be empty or valid -->

  <xsl:template name="cy-120" use-when="$RUN">
    <out>
        <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
          <xsl:where-populated>
            <xsl:for-each select="head(//PoolIdentifier)">
              <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}" type="FmIdentifierInt">
                <xsl:copy-of select="*/*/NONEXISTENT"/>
                <xsl:on-empty select="29"/>
              </xsl:element>
            </xsl:for-each>
          </xsl:where-populated>
      </xsl:source-document>
    </out>
  </xsl:template>

  <!-- within xsl:source-document, use xsl:element: schema-aware, validation by type, on-empty must be empty or valid -->

  <xsl:template name="cy-121" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
          <xsl:where-populated>
            <xsl:for-each select="head(//PoolIdentifier)">
              <xsl:element name="{name(.)}" namespace="{namespace-uri(.)}" type="FmIdentifierInt">
                <xsl:copy-of select="*/*/NONEXISTENT"/>
                <xsl:on-empty select="'NOT_A_NUMBER'"/>
              </xsl:element>
            </xsl:for-each>
          </xsl:where-populated>
      </xsl:source-document>
    </out>
  </xsl:template>



</xsl:transform>
