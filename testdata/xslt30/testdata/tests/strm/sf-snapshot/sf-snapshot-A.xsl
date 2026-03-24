<xsl:transform version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
   

  <!-- within xsl:source-document, use fn:snapshot(): climbing posture -->
  
  <xsl:template name="cy-003" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:variable name="atts" as="attribute(*)*">
          <xsl:sequence select="snapshot(account/transaction[@value &lt; 0]/@value)"/>
        </xsl:variable>
        <xsl:copy-of select="data($atts)"/>  
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use fn:snapshot(): climbing posture -->
  
  <xsl:template name="cy-004" use-when="$RUN">
    <xsl:variable name="extra" as="element()*">
      <PRICE value="101"/>
      <PRICE value="102"/>
    </xsl:variable>
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:variable name="atts" as="attribute(*)*">
          <xsl:sequence select="snapshot((account/transaction[@value &lt; 0]/@value, $extra/@value))"/>
        </xsl:variable>
        <xsl:copy-of select="data($atts)"/> 
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use fn:snapshot(): striding posture, element nodes -->
  
  <xsl:template name="cy-005" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:sequence select="snapshot(/BOOKLIST/BOOKS/ITEM/PRICE)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use fn:snapshot(): striding posture, text nodes -->
  
  <xsl:template name="cy-006" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:sequence select="snapshot(/BOOKLIST/BOOKS/ITEM/PRICE/text())"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use fn:snapshot(): striding posture, element nodes mixed with grounded elements -->
  
  <xsl:template name="cy-008" use-when="$RUN">
    <xsl:variable name="extra" as="element()*">
      <PRICE>100.00</PRICE>
      <PRICE>101.00</PRICE>
    </xsl:variable>
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:sequence select="snapshot(($extra, /BOOKLIST/BOOKS/ITEM/PRICE))"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use fn:snapshot(): descendant text nodes -->
  
  <xsl:template name="cy-009" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <xsl:sequence select="snapshot(//PRICE/text())"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use fn:snapshot(): whole document unchanged -->
  
  <xsl:template name="cy-011" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <head/>
        <xsl:sequence select="snapshot(.)"/>
        <tail/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use fn:snapshot(): copy-namespaces=yes -->
  
  <xsl:template name="cy-021" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:sequence select="snapshot(/*/*:description)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use fn:snapshot(): copy-namespaces="yes", argument grounded and consuming -->
  
  <xsl:template name="cy-025" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:sequence select="snapshot(snapshot(/*/*:description))"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <!-- within xsl:source-document, use fn:snapshot(): argument crawling and consuming, nested nodes -->
  
  <xsl:template name="cy-027" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/nested-numbers.xml">
        <xsl:sequence select="snapshot(descendant::n)"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
</xsl:transform>  