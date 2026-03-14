<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
  
  <!-- streaming, xsl:for-each-group group-adjacent, multi-level grouping -->
  
  <xsl:mode name="s" streamable="yes"/>
    
  <xsl:template name="main">
    <xsl:source-document streamable="yes" href="../docs/transactions.xml">
      <xsl:apply-templates select="account" mode="s"/>
    </xsl:source-document>
  </xsl:template> 


  <xsl:template match="account" mode="s">
     <out>
      <xsl:for-each-group select="transaction"
         group-adjacent="format-date(xs:date(@date), '[W]')"
         composite="no">
         <week nr="{current-grouping-key()}">
           <xsl:for-each-group select="current-group()" group-adjacent="xs:date(@date)">
             <day date="{current-grouping-key()}">
                <xsl:copy-of select="current-group()"/>
             </day>
           </xsl:for-each-group>
         </week>
      </xsl:for-each-group> 
    </out>
  </xsl:template>   
  

       
</xsl:transform>

