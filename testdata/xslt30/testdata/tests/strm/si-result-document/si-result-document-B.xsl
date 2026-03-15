<xsl:transform version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:err="http://www.w3.org/2005/xqt-errors"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs err">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
    
  <xsl:import-schema schema-location="../docs/loans.xsd"/>  
   
  <!-- within xsl:source-document, use xsl:result-document: schema-aware, validation=strict -->
  
  <xsl:template name="d-101" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:result-document href="d-101.xml" validation="strict">
            <xsl:copy-of select="child::node()"/>
          </xsl:result-document>
        </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:result-document: schema-aware, validation=lax -->
  
  <xsl:template name="d-102" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
          <xsl:result-document href="d-102.xml" validation="lax">
            <xsl:copy-of select="child::node()"/>
          </xsl:result-document>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:result-document: schema-aware, validation=strip -->
  
  <xsl:template name="d-103" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
          <xsl:result-document href="d-103.xml" validation="strip">
            <xsl:copy-of select="child::node()"/>
          </xsl:result-document>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:result-document: schema-aware, validation=preserve -->
  
  <xsl:template name="d-104" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
          <xsl:result-document href="d-104.xml" validation="preserve">
            <xsl:copy-of select="child::node()"/>
          </xsl:result-document>
      </xsl:source-document>
    </out>
  </xsl:template>  
  
  <!-- within xsl:source-document, use xsl:result-document: schema-aware, validation by type -->
  
  <xsl:template name="d-105" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
          <xsl:result-document href="d-105.xml" type="myroot">
            <xsl:copy-of select="child::node()"/>
          </xsl:result-document>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:result-document: schema-aware, validation=strict, no element declaration available -->
  
  <xsl:template name="d-106" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
          <xsl:result-document href="d-106.xml" validation="strict">
            <xsl:copy-of select="child::node()"/>
          </xsl:result-document>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:result-document: schema-aware, validation=lax, no element declaration available -->
  
  <xsl:template name="d-107" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
          <xsl:result-document href="d-107.xml" validation="lax">
            <xsl:copy-of select="child::node()"/>
          </xsl:result-document>
      </xsl:source-document>
    </out>
  </xsl:template> 
  
  <!-- within xsl:source-document, use xsl:result-document: schema-aware, validation by type, invalid (wrong namespace) -->
  
  <xsl:template name="d-108" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
          <xsl:result-document href="d-108.xml" validation="strict">
            <xsl:copy-of select="child::node()"/>
          </xsl:result-document>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:result-document: schema-aware, validation=strict, no element declaration available, caught -->
  
  <xsl:template name="d-109" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:try>
          <xsl:result-document href="d-109.xml" validation="strict">
            <xsl:copy-of select="child::node()"/>
          </xsl:result-document>
          <xsl:catch errors="*">
            <xsl:value-of select="local-name-from-QName($err:code)"/>
          </xsl:catch>
        </xsl:try>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:result-document: schema-aware, validation by type, invalid, caught -->
  
  <xsl:template name="d-110" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
        <xsl:try>
          <xsl:result-document href="d-110.xml" type="myroot">
            <xsl:copy-of select="child::node()"/>
          </xsl:result-document>
          <xsl:catch errors="*">
            <xsl:value-of select="local-name-from-QName($err:code)"/>
          </xsl:catch>
        </xsl:try>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:result-document: schema-aware, validation=strict, argument is consuming and grounded -->
  
  <xsl:template name="d-111" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
          <xsl:for-each select="/*/*">
            <xsl:result-document href="d-111.xml" validation="strict">
              <xsl:copy-of select="."/>
            </xsl:result-document>
          </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:result-document: schema-aware, validation=lax, argument is consuming and grounded -->
  
  <xsl:template name="d-112" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
          <xsl:for-each select="/*/*">
            <xsl:result-document href="d-112.xml" validation="lax">
              <xsl:copy-of select="."/>
            </xsl:result-document>
          </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:result-document: schema-aware, validation=strip, argument is consuming and grounded -->
  
  <xsl:template name="d-113" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
          <xsl:for-each select="copy-of(/*/*)">
            <xsl:result-document href="d-113.xml" validation="strip">
              <xsl:copy-of select="child::node()"/>
            </xsl:result-document>
          </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:result-document: schema-aware, validation=preserve, argument is consuming and grounded -->
  
  <xsl:template name="d-114" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
         <xsl:for-each select="copy-of(/*/*)">
            <xsl:result-document href="d-114.xml" validation="preserve">
              <xsl:copy-of select="child::node()"/>
            </xsl:result-document>
          </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>  
  
  <!-- within xsl:source-document, use xsl:result-document: schema-aware, validation by type, argument is consuming and grounded -->
  
  <xsl:template name="d-115" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
          <xsl:for-each select="/*/*">
            <xsl:result-document href="d-115.xml" type="ACERequest">
              <xsl:copy-of select="."/>
            </xsl:result-document>
          </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:result-document: schema-aware, validate document node by type, success -->
  
  <xsl:template name="d-116" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:result-document href="d-116.xml" type="xs:decimal">
          <in><xsl:value-of select="head(//@version)"/></in>
        </xsl:result-document>
      </xsl:source-document>
    </out>
  </xsl:template> 
  
  <!-- within xsl:source-document, use xsl:result-document: schema-aware, validate document node by type, failure -->
  
  <xsl:template name="d-117" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
        <xsl:result-document href="d-117.xml" type="xs:date">
          <in><xsl:value-of select="count(//*)"/></in>
        </xsl:result-document>
      </xsl:source-document>
    </out>
  </xsl:template>
  
<!-- within xsl:source-document, use xsl:result-document: schema-aware, validation=strict, element node -->
  
  <xsl:template name="d-121" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
          <xsl:result-document href="d-121.xml" validation="strict">
            <xsl:copy-of select="child::node()"/>
          </xsl:result-document>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:result-document: schema-aware, validation=lax, element node -->
  
  <xsl:template name="d-122" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
          <xsl:result-document href="d-122.xml" validation="lax">
            <xsl:copy-of select="child::node()"/>
          </xsl:result-document>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:result-document: schema-aware, validation=strip, element node -->
  
  <xsl:template name="d-123" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
          <xsl:result-document href="d-123.xml" validation="strip">
            <xsl:copy-of select="child::node()"/>
          </xsl:result-document>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:result-document: schema-aware, validation=preserve, element node -->
  
  <xsl:template name="d-124" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
          <xsl:result-document href="d-124.xml" validation="preserve">
            <xsl:copy-of select="child::node()"/>
          </xsl:result-document>
      </xsl:source-document>
    </out>
  </xsl:template>  
  
  <!-- within xsl:source-document, use xsl:result-document: schema-aware, validation by type, element node -->
  
  <xsl:template name="d-125" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans-noNS.xml">
          <xsl:result-document href="d-125.xml" type="myroot">
            <xsl:copy-of select="child::node()"/>
          </xsl:result-document>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:result-document: schema-aware, validation=strict, element node, no element declaration available -->
  
  <xsl:template name="d-126" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
          <xsl:result-document href="d-126.xml" validation="strict">
            <xsl:copy-of select="child::node()"/>
          </xsl:result-document>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:result-document: schema-aware, validation=lax, element node, no element declaration available -->
  
  <xsl:template name="d-127" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
          <xsl:result-document href="d-127.xml" validation="lax">
            <xsl:copy-of select="child::node()"/>
          </xsl:result-document>
      </xsl:source-document>
    </out>
  </xsl:template> 
  
  <!-- within xsl:source-document, use xsl:result-document: schema-aware, validation by type, element node, invalid (wrong namespace) -->
  
  <xsl:template name="d-128" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/loans.xml">
          <xsl:result-document href="d-128.xml" type="myroot">
            <xsl:copy-of select="child::node()"/>
          </xsl:result-document>
      </xsl:source-document>
    </out>
  </xsl:template>    
   
  
  
</xsl:transform>  