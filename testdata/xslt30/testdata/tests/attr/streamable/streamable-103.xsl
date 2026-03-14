<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
  
     
    <!-- within a streaming template, use xsl:for-each-group group-adjacent -->
    
    <xsl:mode name="s" streamable="yes"/>
         
    <xsl:output method="xml" indent="yes" encoding="UTF-8" />
      
    <xsl:template name="main">
       <xsl:source-document streamable="true" href="transactions.xml">
        <xsl:apply-templates select="account" mode="s"/>
      </xsl:source-document>
    </xsl:template> 
  
  
    <xsl:template match="account" mode="s">
       <out>
        <xsl:for-each-group select="transaction"
           group-adjacent="@date">
           <batch date="{current-grouping-key()}">
              <xsl:sequence select="current-group()"/>
           </batch>
        </xsl:for-each-group> 
      </out>
    </xsl:template>   
    

       
</xsl:transform>

