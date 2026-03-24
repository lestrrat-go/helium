<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">
       
  
  <!-- drill-down expression is xsl:variable select="." as="xs:integer" -->
   
  <xsl:import-schema schema-location="loans.xsd"/>

  <xsl:mode name="s" streamable="yes"/>
  <xsl:mode name="t" streamable="yes"/>
       
  <xsl:output method="xml" encoding="UTF-8" />
  
  <xsl:param name="use-prefix" select="true()"/>
     
  <xsl:template name="main" match="/">
    <out>
      <xsl:source-document streamable="true" href="loans.xml">
        <xsl:apply-templates select="." mode="s"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <xsl:template match="text()" mode="s"/>
       
   <xsl:template match="your:extra" mode="s" xmlns:your="http://your.com/ns">
     <extra>
       <xsl:variable name="n" as="xs:integer" select="."/>
       <xsl:value-of select="$n gt 20"/>
     </extra>
   </xsl:template>
   
   

</xsl:transform>

