<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:g="http://www.w3.org/xsl-tests/grouped-transactions"
    exclude-result-prefixes=" xs"
    version="3.0">
       
   
   <!-- within a streaming template, compute average of a list-valued attribute value -->
    
   <xsl:mode streamable="yes"/>
        
   <xsl:output method="xml" indent="yes" encoding="UTF-8" />
   
  <xsl:import-schema namespace="http://www.w3.org/xsl-tests/grouped-transactions" schema-location="grouped-transactions.xsd"/>
 
   <xsl:strip-space elements="*"/>
     
   <xsl:template name="main">
     <out>
       <xsl:source-document streamable="true" href="grouped-transactions.xml" validation="strict">
         <xsl:apply-templates select="."/>
       </xsl:source-document>
     </out>
   </xsl:template>
    
   <xsl:template match="*">
     <xsl:copy>
       <xsl:copy-of select="@*"/>
       <xsl:apply-templates/>
     </xsl:copy>
   </xsl:template>
   
   <xsl:template match="g:account">
     <account avg="{round-half-to-even(avg(g:transaction/@value), 2)}"/>
   </xsl:template>
    
</xsl:transform>

