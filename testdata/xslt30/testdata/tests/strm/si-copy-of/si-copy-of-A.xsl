<xsl:transform version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
   
  <!-- within xsl:source-document, use xsl:copy-of: atomic values, consuming -->
  
  <xsl:template name="cy-001" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:copy-of select="data(account/transaction[@value &lt; 0]/@value)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy-of: atomic values, consuming and non-consuming -->
  
  <xsl:template name="cy-002" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:copy-of select="data(account/transaction[@value &lt; 0]/@value), 101, 102"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy-of: climbing posture -->
  
  <xsl:template name="cy-003" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:variable name="atts" as="attribute(*)*">
          <xsl:copy-of select="account/transaction[@value &lt; 0]/@value"/>
        </xsl:variable>
        <xsl:copy-of select="data($atts)"/>  
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy-of: climbing posture -->
  
  <xsl:template name="cy-004" use-when="$RUN">
    <xsl:variable name="extra" as="element()*">
      <PRICE value="101"/>
      <PRICE value="102"/>
    </xsl:variable>
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:variable name="atts" as="attribute(*)*">
          <xsl:copy-of select="account/transaction[@value &lt; 0]/@value, $extra/@value"/>
        </xsl:variable>
        <xsl:copy-of select="data($atts)"/> 
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy-of: striding posture, element nodes -->
  
  <xsl:template name="cy-005" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:copy-of select="/BOOKLIST/BOOKS/ITEM/PRICE"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy-of: striding posture, text nodes -->
  
  <xsl:template name="cy-006" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:copy-of select="/BOOKLIST/BOOKS/ITEM/PRICE/text()"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy-of: striding posture, text nodes mixed with atomic values -->
  
  <xsl:template name="cy-007" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:copy-of select="/BOOKLIST/BOOKS/ITEM/PRICE/text(), 101, 102"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy-of: striding posture, element nodes mixed with grounded elements -->
  
  <xsl:template name="cy-008" use-when="$RUN">
    <xsl:variable name="extra" as="element()*">
      <PRICE>100.00</PRICE>
      <PRICE>101.00</PRICE>
    </xsl:variable>
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:copy-of select="$extra, /BOOKLIST/BOOKS/ITEM/PRICE"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy-of: descendant text nodes -->
  
  <xsl:template name="cy-009" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:copy-of select="//PRICE/text()"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy-of: descendant text nodes mixed with atomic values -->
  
  <xsl:template name="cy-010" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:copy-of select="100, 101, //PRICE/text()"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy-of: whole document unchanged -->
  
  <xsl:template name="cy-011" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <head/>
        <xsl:copy-of select="."/>
        <tail/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy-of: copy-namespaces=no -->
  
  <xsl:template name="cy-020" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:copy-of select="/*/*:description" copy-namespaces="no"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy-of: copy-namespaces=yes -->
  
  <xsl:template name="cy-021" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:copy-of select="/*/*:description" copy-namespaces="yes"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy-of: validation="strip" (non-schema-aware) -->
  
  <xsl:template name="cy-022" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:copy-of select="/*/*:description" copy-namespaces="no" validation="strip"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy-of: validation="preserve" (non-schema-aware) -->
  
  <xsl:template name="cy-023" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:copy-of select="/*/*:description" copy-namespaces="no" validation="preserve"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy-of: validation="lax" (non-schema-aware) -->
  
  <xsl:template name="cy-024" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:copy-of select="/*/*:description" copy-namespaces="no" validation="lax"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy-of: copy-namespaces="yes", argument grounded and consuming -->
  
  <xsl:template name="cy-025" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:copy-of select="copy-of(/*/*:description)" copy-namespaces="yes"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy-of: copy-namespaces="no", argument grounded and consuming -->
  
  <xsl:template name="cy-026" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:copy-of select="copy-of(/*/*:description)" copy-namespaces="no"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use xsl:copy-of: argument crawling and consuming, nested nodes -->
  
  <xsl:template name="cy-027" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/nested-numbers.xml">
        <xsl:copy-of select="descendant::n"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
</xsl:transform>  