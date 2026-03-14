<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
   
  <!-- within xsl:source-document, use xsl:for-each-group group-adjacent -->
  
  <xsl:mode name="s" streamable="yes"/>
       
    
  <xsl:template name="main">
    <xsl:source-document streamable="yes" href="../docs/transactions.xml">
     <out>
      <xsl:for-each-group select="account/transaction" group-adjacent="@date">
         <batch date="{current-grouping-key()}">
            <xsl:sequence select="current-group()"/>
         </batch>
      </xsl:for-each-group> 
     </out>
    </xsl:source-document>
  </xsl:template> 


</xsl:transform>

