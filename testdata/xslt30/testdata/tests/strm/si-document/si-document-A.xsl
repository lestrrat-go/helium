<xsl:transform version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
   
  <!-- within xsl:source-document, use xsl:document: atomic values, consuming -->
  
  <xsl:template name="d-001" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:for-each select="account/transaction[@value &lt; 0]/@value">
          <xsl:document><xsl:sequence select="data(.)"/></xsl:document>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: atomic values, consuming and non-consuming -->
  
  <xsl:template name="d-002" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:document>
          <xsl:for-each select="data(account/transaction[@value &lt; 0]/@value), 101, 102">
            <xsl:sequence select="data(.)"/>
          </xsl:for-each>  
    	</xsl:document> 
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: climbing posture -->
  
  <xsl:template name="d-003" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:variable name="docs" as="document-node()*">
          <xsl:for-each select="account/transaction[@value &lt; 0]/@value">
            <xsl:document><xsl:sequence select="data(.)"/></xsl:document>
          </xsl:for-each>
        </xsl:variable>
        <xsl:copy-of select="data($docs)"/>  
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: climbing posture -->
  
  <xsl:template name="d-004" use-when="$RUN">
    <xsl:variable name="extra" as="element()*">
      <PRICE value="101"/>
      <PRICE value="102"/>
    </xsl:variable>
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:variable name="docs" as="document-node()*">
          <xsl:for-each select="account/transaction[@value &lt; 0]/@value, $extra/@value">
            <xsl:document><xsl:sequence select="data(.)"/></xsl:document>
          </xsl:for-each>
        </xsl:variable>
        <xsl:copy-of select="data($docs)"/> 
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: striding posture, element nodes -->
  
  <xsl:template name="d-005" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="/BOOKLIST/BOOKS/ITEM/PRICE">
          <xsl:document>
            <xsl:copy-of select="."/>
          </xsl:document>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: striding posture, text nodes -->
  
  <xsl:template name="d-006" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="/BOOKLIST/BOOKS/ITEM/PRICE/text()">
          <xsl:document>
            <xsl:copy-of select="."/>
          </xsl:document>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: striding posture, text nodes mixed with atomic values -->
  
  <xsl:template name="d-007" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="/BOOKLIST/BOOKS/ITEM/PRICE/text(), 101, 102">
          <xsl:document>
            <xsl:copy-of select="."/>
          </xsl:document>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: striding posture, element nodes mixed with grounded elements -->
  
  <xsl:template name="d-008" use-when="$RUN">
    <xsl:variable name="extra" as="element()*">
      <PRICE>100.00</PRICE>
      <PRICE>101.00</PRICE>
    </xsl:variable>
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="$extra, /BOOKLIST/BOOKS/ITEM/PRICE">
          <xsl:document>
            <xsl:copy-of select="."/>
          </xsl:document>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: descendant text nodes -->
  
  <xsl:template name="d-009" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="outermost(//PRICE)">
          <xsl:document>
            <xsl:sequence select="text()"/>
          </xsl:document>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: descendant text nodes mixed with atomic values -->
  
  <xsl:template name="d-010" use-when="true() or $RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:for-each select="100, 101, /BOOKLIST/BOOKS/ITEM/PRICE/text()">
          <xsl:document>
            <xsl:sequence select="."/>
          </xsl:document>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: whole document unchanged -->
  
  <xsl:template name="d-011" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <head/>
        <xsl:document>
          <xsl:copy-of select="child::node()"/>
        </xsl:document>
        <tail/>
      </xsl:source-document>
    </out>
  </xsl:template>
  

  <!-- within xsl:source-document, use xsl:document: validation="strip" (non-schema-aware) -->
  
  <xsl:template name="d-022" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:for-each select="/*/*:description">
          <xsl:document validation="strip">
            <xsl:copy-of select="."/>
          </xsl:document>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: validation="preserve" (non-schema-aware) -->
  
  <xsl:template name="d-023" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:for-each select="/*/*:description">
          <xsl:document validation="preserve">
            <xsl:copy-of select="."/>
          </xsl:document>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:document: validation="lax" (non-schema-aware) -->
  
  <xsl:template name="d-024" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:for-each select="/*/*:description">
          <xsl:document validation="lax">
            <xsl:copy-of select="."/>
          </xsl:document>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  

  
</xsl:transform>  