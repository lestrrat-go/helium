<xsl:transform version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:err="http://www.w3.org/2005/xqt-errors"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs err">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
    
  <xsl:import-schema schema-location="../docs/loans.xsd"/>  
   
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validation=strict -->
  
  <xsl:template name="cy-101" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy">
          <xsl:copy validation="strict">
            <xsl:copy-of select="child::node()"/>
          </xsl:copy>
        </xsl:variable>
        <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validation=lax -->
  
  <xsl:template name="cy-102" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy">
          <xsl:copy select="." validation="lax">
            <xsl:copy-of select="child::node()"/>
          </xsl:copy>
        </xsl:variable>
        <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validation=strip -->
  
  <xsl:template name="cy-103" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy">
          <xsl:copy select="." validation="strip">
            <xsl:copy-of select="child::node()"/>
          </xsl:copy>
        </xsl:variable>
        <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validation=preserve -->
  
  <xsl:template name="cy-104" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy">
          <xsl:copy select="." validation="preserve">
            <xsl:copy-of select="child::node()"/>
          </xsl:copy>
        </xsl:variable>
        <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>  
  
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validation by type -->
  
  <xsl:template name="cy-105" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy">
          <xsl:copy select="." type="myroot">
            <xsl:copy-of select="child::node()"/>
          </xsl:copy>
        </xsl:variable>
        <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validation=strict, no element declaration available -->
  
  <xsl:template name="cy-106" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:variable name="copy">
          <xsl:copy select="." validation="strict">
            <xsl:copy-of select="child::node()"/>
          </xsl:copy>
        </xsl:variable>
        <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validation=lax, no element declaration available -->
  
  <xsl:template name="cy-107" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:variable name="copy">
          <xsl:copy select="." validation="lax">
            <xsl:copy-of select="child::node()"/>
          </xsl:copy>
        </xsl:variable>
        <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template> 
  
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validation by type, invalid (wrong namespace) -->
  
  <xsl:template name="cy-108" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:variable name="copy">
          <xsl:copy select="." validation="strict">
            <xsl:copy-of select="child::node()"/>
          </xsl:copy>
        </xsl:variable>
        <xsl:value-of select="$copy/* instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validation=strict, no element declaration available, caught -->
  
  <xsl:template name="cy-109" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:try>
          <xsl:copy select="." validation="strict">
            <xsl:copy-of select="child::node()"/>
          </xsl:copy>
          <xsl:catch errors="*">
            <xsl:value-of select="local-name-from-QName($err:code)"/>
          </xsl:catch>
        </xsl:try>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validation by type, invalid, caught -->
  
  <xsl:template name="cy-110" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:try>
          <xsl:copy select="." type="myroot">
            <xsl:copy-of select="child::node()"/>
          </xsl:copy>
          <xsl:catch errors="*">
            <xsl:value-of select="local-name-from-QName($err:code)"/>
          </xsl:catch>
        </xsl:try>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validation=strict, argument is consuming and grounded -->
  
  <xsl:template name="cy-111" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:for-each select="copy-of(/*/*)">
            <xsl:copy validation="strict">
              <xsl:copy-of select="child::node()"/>
            </xsl:copy>
          </xsl:for-each>
        </xsl:variable>
        <xsl:value-of select="$copy instance of element(*, ACERequest)*"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validation=lax, argument is consuming and grounded -->
  
  <xsl:template name="cy-112" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:for-each select="copy-of(/*/*)">
            <xsl:copy validation="lax">
              <xsl:copy-of select="child::node()"/>
            </xsl:copy>
          </xsl:for-each>
        </xsl:variable>
        <xsl:value-of select="$copy instance of element(*, ACERequest)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validation=strip, argument is consuming and grounded -->
  
  <xsl:template name="cy-113" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:for-each select="copy-of(/*/*)">
            <xsl:copy validation="strip">
              <xsl:copy-of select="child::node()"/>
            </xsl:copy>
          </xsl:for-each>
        </xsl:variable>
        <xsl:value-of select="$copy instance of element(*, ACERequest)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validation=preserve, argument is consuming and grounded -->
  
  <xsl:template name="cy-114" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:for-each select="copy-of(/*/*)">
            <xsl:copy validation="preserve">
              <xsl:copy-of select="child::node()"/>
            </xsl:copy>
          </xsl:for-each>
        </xsl:variable>
        <xsl:value-of select="$copy instance of element(*, ACERequest)"/>
      </xsl:source-document>
    </out>
  </xsl:template>  
  
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validation by type, argument is consuming and grounded -->
  
  <xsl:template name="cy-115" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:for-each select="copy-of(/*/*)">
            <xsl:copy type="ACERequest">
              <xsl:copy-of select="child::node()"/>
            </xsl:copy>
          </xsl:for-each>
        </xsl:variable>
        <xsl:value-of select="$copy instance of element(*, ACERequest)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validate attribute node by type, success -->
  
  <xsl:template name="cy-116" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:copy select="/*/*/@version" type="xs:decimal"/>
      </xsl:source-document>
    </out>
  </xsl:template> 
  
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validate attribute node by type, failure -->
  
  <xsl:template name="cy-117" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:copy select="/*/*/@version" type="xs:date"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validation=strict, element node -->
  
  <xsl:template name="cy-121" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:copy select="*" validation="strict">
            <xsl:copy-of select="child::node()"/>
          </xsl:copy>
        </xsl:variable>
        <xsl:value-of select="$copy instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validation=lax, element node -->
  
  <xsl:template name="cy-122" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:copy select="*" validation="lax">
            <xsl:copy-of select="child::node()"/>
          </xsl:copy>
        </xsl:variable>
        <xsl:value-of select="$copy instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validation=strip, element node -->
  
  <xsl:template name="cy-123" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:copy select="*" validation="strip">
            <xsl:copy-of select="child::node()"/>
          </xsl:copy>
        </xsl:variable>
        <xsl:value-of select="$copy instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validation=preserve, element node -->
  
  <xsl:template name="cy-124" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:copy select="*" validation="preserve">
            <xsl:copy-of select="child::node()"/>
          </xsl:copy>
        </xsl:variable>
        <xsl:value-of select="$copy instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>  
  
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validation by type, element node -->
  
  <xsl:template name="cy-125" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:copy select="*" type="myroot">
            <xsl:copy-of select="child::node()"/>
          </xsl:copy>
        </xsl:variable>
        <xsl:value-of select="$copy instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validation=strict, element node, no element declaration available -->
  
  <xsl:template name="cy-126" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:copy select="*" validation="strict">
            <xsl:copy-of select="child::node()"/>
          </xsl:copy>
        </xsl:variable>
        <xsl:value-of select="$copy instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validation=lax, element node, no element declaration available -->
  
  <xsl:template name="cy-127" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:copy select="*" validation="lax">
            <xsl:copy-of select="child::node()"/>
          </xsl:copy>
        </xsl:variable>
        <xsl:value-of select="$copy instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template> 
  
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validation by type, element node, invalid (wrong namespace) -->
  
  <xsl:template name="cy-128" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:variable name="copy" as="node()">
          <xsl:copy select="*" validation="strict">
            <xsl:copy-of select="child::node()"/>
          </xsl:copy>
        </xsl:variable>
        <xsl:value-of select="$copy instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy: schema-aware, validation=strict, element node, passes xsl:variable/@as -->
  
  <xsl:template name="cy-129" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="element(*, myroot)">
          <xsl:copy select="*" validation="strict">
            <xsl:copy-of select="child::node()"/>
          </xsl:copy>
        </xsl:variable>
        <xsl:value-of select="$copy instance of element(*, myroot)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
<!-- within xsl:source-document, use xsl:copy: schema-aware, validation=strict, element node, fails xsl:variable/@as -->
  
  <xsl:template name="cy-130" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:variable name="copy" as="element(*, xs:integer)">
          <xsl:copy select="*" validation="strict">
            <xsl:copy-of select="child::node()"/>
          </xsl:copy>
        </xsl:variable>
        <xsl:value-of select="empty($copy/comment())"/>
      </xsl:source-document>
    </out>
  </xsl:template>        
   
  
  
</xsl:transform>  