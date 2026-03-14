<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:g="http://www.w3.org/xsl-tests/grouped-transactions"
    exclude-result-prefixes="g xs"
    version="3.0">
       
  
   <!-- within a streaming template, use positional predicate in select expression -->
    
   <xsl:import-schema namespace="http://www.w3.org/xsl-tests/grouped-transactions" schema-location="grouped-transactions.xsd"/>
 
   <xsl:mode name="s" streamable="yes"/>
        
   <xsl:output method="xml" indent="no" encoding="UTF-8" />
    
   <xsl:template name="main" match="/">
     <out>
       <xsl:source-document streamable="true" href="grouped-transactions.xml" validation="strict">
         <xsl:apply-templates select="." mode="s"/>
       </xsl:source-document>
     </out>
   </xsl:template>
   
   <xsl:template match="g:account" mode="s">
     <xsl:apply-templates select="*[2]" mode="s"/>
   </xsl:template>
   
   <xsl:template match="g:transaction" mode="s">
     <max date="{@date}"><xsl:value-of select="format-number(max(@value), '0.00')"/></max>
   </xsl:template>
    
   <xsl:template match="text()" mode="s"/>
  
    
</xsl:transform>

