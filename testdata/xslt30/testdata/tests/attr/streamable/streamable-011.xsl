<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    version="3.0">
       
  
  <!-- Test that attributes, namespaces, comments, and PIs are copied in streaming mode -->
   
  <xsl:import-schema schema-location="loans.xsd"/>

  <xsl:mode name="s" streamable="yes"/>

  
  <xsl:param name="use-prefix" select="true()"/>
    
    
  <xsl:template name="main" match="/">
    <out>
      <xsl:source-document streamable="true" href="loans.xml">
        <xsl:apply-templates select="." mode="s"/>
      </xsl:source-document>
    </out>
  </xsl:template>
  
  <xsl:template match="text()" mode="s"/>
  
      
   <xsl:template match="Loan" mode="s">
     <xsl:variable name="prefix" select="if ($use-prefix) then '#' else ''"/>
     <xsl:variable name="Loan">
       <xsl:copy-of select="."/>
     </xsl:variable>
      <Loan prefix="{$prefix}">
        <xsl:copy-of select="$Loan//your:extra" xmlns:your="http://your.com/ns"/>
      </Loan>
   </xsl:template>
   

</xsl:transform>

