<xsl:transform version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:err="http://www.w3.org/2005/xqt-errors"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs err">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
    
  <xsl:import-schema schema-location="../docs/loans.xsd"/>  
   
  <!-- within xsl:source-document, use xsl:document: schema-aware, validation=strict -->
  
  <xsl:template name="d-101" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy">
          <xsl:document validation="strict">
            <xsl:copy-of select="child::node()"/>
          </xsl:document>
        </xsl:variable>
        <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: schema-aware, validation=lax -->
  
  <xsl:template name="d-102" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy">
          <xsl:document validation="lax">
            <xsl:copy-of select="child::node()"/>
          </xsl:document>
        </xsl:variable>
        <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: schema-aware, validation=strip -->
  
  <xsl:template name="d-103" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy">
          <xsl:document validation="strip">
            <xsl:copy-of select="child::node()"/>
          </xsl:document>
        </xsl:variable>
        <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: schema-aware, validation=preserve -->
  
  <xsl:template name="d-104" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy">
          <xsl:document validation="preserve">
            <xsl:copy-of select="child::node()"/>
          </xsl:document>
        </xsl:variable>
        <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>  
  
  <!-- within xsl:source-document, use xsl:document: schema-aware, validation by type -->
  
  <xsl:template name="d-105" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy">
          <xsl:document type="myroot">
            <xsl:copy-of select="child::node()"/>
          </xsl:document>
        </xsl:variable>
        <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: schema-aware, validation=strict, no element declaration available -->
  
  <xsl:template name="d-106" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:variable name="copy">
          <xsl:document validation="strict">
            <xsl:copy-of select="child::node()"/>
          </xsl:document>
        </xsl:variable>
        <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: schema-aware, validation=lax, no element declaration available -->
  
  <xsl:template name="d-107" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:variable name="copy">
          <xsl:document validation="lax">
            <xsl:copy-of select="child::node()"/>
          </xsl:document>
        </xsl:variable>
        <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template> 
  
  <!-- within xsl:source-document, use xsl:document: schema-aware, validation by type, invalid (wrong namespace) -->
  
  <xsl:template name="d-108" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:variable name="copy">
          <xsl:document validation="strict">
            <xsl:copy-of select="child::node()"/>
          </xsl:document>
        </xsl:variable>
        <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: schema-aware, validation=strict, no element declaration available, caught -->
  
  <xsl:template name="d-109" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:try>
          <xsl:document validation="strict">
            <xsl:copy-of select="child::node()"/>
          </xsl:document>
          <xsl:catch errors="*">
            <xsl:value-of select="local-name-from-QName($err:code)"/>
          </xsl:catch>
        </xsl:try>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: schema-aware, validation by type, invalid, caught -->
  
  <xsl:template name="d-110" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:try>
          <xsl:document type="myroot">
            <xsl:copy-of select="child::node()"/>
          </xsl:document>
          <xsl:catch errors="*">
            <xsl:value-of select="local-name-from-QName($err:code)"/>
          </xsl:catch>
        </xsl:try>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: schema-aware, validation=strict, argument is consuming and grounded -->
  
  <xsl:template name="d-111" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:for-each select="/*/*">
            <xsl:document validation="strict">
              <xsl:copy-of select="."/>
            </xsl:document>
          </xsl:for-each>
        </xsl:variable>
        <xsl:value-of select="$copy instance of document-node(element(*, ACERequest))*"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: schema-aware, validation=lax, argument is consuming and grounded -->
  
  <xsl:template name="d-112" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:for-each select="/*/*">
            <xsl:document validation="lax">
              <xsl:copy-of select="."/>
            </xsl:document>
          </xsl:for-each>
        </xsl:variable>
        <xsl:value-of select="$copy instance of document-node(element(*, ACERequest))*"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: schema-aware, validation=strip, argument is consuming and grounded -->
  
  <xsl:template name="d-113" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:for-each select="copy-of(/*/*)">
            <xsl:document validation="strip">
              <xsl:copy-of select="child::node()"/>
            </xsl:document>
          </xsl:for-each>
        </xsl:variable>
        <xsl:value-of select="$copy instance of document-node(element(*, ACERequest))*"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: schema-aware, validation=preserve, argument is consuming and grounded -->
  
  <xsl:template name="d-114" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:for-each select="copy-of(/*/*)">
            <xsl:document validation="preserve">
              <xsl:copy-of select="child::node()"/>
            </xsl:document>
          </xsl:for-each>
        </xsl:variable>
        <xsl:value-of select="$copy instance of document-node(element(*, ACERequest))"/>
      </xsl:source-document>
    </out>
  </xsl:template>  
  
  <!-- within xsl:source-document, use xsl:document: schema-aware, validation by type, argument is consuming and grounded -->
  
  <xsl:template name="d-115" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:for-each select="/*/*">
            <xsl:document type="ACERequest">
              <xsl:copy-of select="."/>
            </xsl:document>
          </xsl:for-each>
        </xsl:variable>
        <xsl:value-of select="$copy instance of document-node(element(*, ACERequest))"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: schema-aware, validate document node by type, success -->
  
  <xsl:template name="d-116" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:document type="xs:decimal">
          <in><xsl:value-of select="head(//@version)"/></in>
        </xsl:document>
      </xsl:source-document>
    </out>
  </xsl:template> 
  
  <!-- within xsl:source-document, use xsl:document: schema-aware, validate document node by type, failure -->
  
  <xsl:template name="d-117" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:document type="xs:date">
          <in><xsl:value-of select="count(//*)"/></in>
        </xsl:document>
      </xsl:source-document>
    </out>
  </xsl:template>
  
<!-- within xsl:source-document, use xsl:document: schema-aware, validation=strict, element node -->
  
  <xsl:template name="d-121" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:document validation="strict">
            <xsl:copy-of select="child::node()"/>
          </xsl:document>
        </xsl:variable>
        <xsl:value-of select="$copy instance of document-node(element(*, myroot))"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: schema-aware, validation=lax, element node -->
  
  <xsl:template name="d-122" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:document validation="lax">
            <xsl:copy-of select="child::node()"/>
          </xsl:document>
        </xsl:variable>
        <xsl:value-of select="$copy instance of document-node(element(*, myroot))"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: schema-aware, validation=strip, element node -->
  
  <xsl:template name="d-123" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:document validation="strip">
            <xsl:copy-of select="child::node()"/>
          </xsl:document>
        </xsl:variable>
        <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: schema-aware, validation=preserve, element node -->
  
  <xsl:template name="d-124" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:document validation="preserve">
            <xsl:copy-of select="child::node()"/>
          </xsl:document>
        </xsl:variable>
        <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>  
  
  <!-- within xsl:source-document, use xsl:document: schema-aware, validation by type, element node -->
  
  <xsl:template name="d-125" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:document type="myroot">
            <xsl:copy-of select="child::node()"/>
          </xsl:document>
        </xsl:variable>
        <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: schema-aware, validation=strict, element node, no element declaration available -->
  
  <xsl:template name="d-126" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:document validation="strict">
            <xsl:copy-of select="child::node()"/>
          </xsl:document>
        </xsl:variable>
        <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: schema-aware, validation=lax, element node, no element declaration available -->
  
  <xsl:template name="d-127" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:document validation="lax">
            <xsl:copy-of select="child::node()"/>
          </xsl:document>
        </xsl:variable>
        <xsl:value-of select="$copy instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template> 
  
  <!-- within xsl:source-document, use xsl:document: schema-aware, validation by type, element node, invalid (wrong namespace) -->
  
  <xsl:template name="d-128" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:document type="myroot">
            <xsl:copy-of select="child::node()"/>
          </xsl:document>
        </xsl:variable>
        <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>    
   
  
  
</xsl:transform>  