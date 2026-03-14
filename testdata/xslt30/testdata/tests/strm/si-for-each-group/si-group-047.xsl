<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
   
  <!-- within a streaming template, use xsl:for-each-group group-adjacent, group on attribute nodes -->
  
  <xsl:mode name="s" streamable="yes"/>
       
    
  <xsl:template name="main">
    <xsl:source-document streamable="yes" href="../docs/transactions.xml">
    	<xsl:apply-templates select="account" mode="s"/>
    </xsl:source-document>
  </xsl:template> 


  <xsl:template match="account" mode="s">
     <out>
      <xsl:for-each-group select="transaction" group-adjacent="@date">
         <xsl:choose>
           <xsl:when test="day-from-date(xs:date(current-grouping-key())) mod 2 = 0">
             <even date="{current-grouping-key()}" position="{position()}">
               <xsl:for-each select="current-group()">
                 <value position="{position()}"><xsl:value-of select="@value"/></value>
               </xsl:for-each>
             </even>
           </xsl:when>
           <xsl:otherwise>
             <odd date="{current-grouping-key()}" position="{position()}">
               <xsl:for-each select="current-group()">
                 <value position="{position()}"><xsl:value-of select="@value"/></value>
               </xsl:for-each>
             </odd>
           </xsl:otherwise>
         </xsl:choose>
      </xsl:for-each-group> 
    </out>
  </xsl:template>   
       
</xsl:transform>

