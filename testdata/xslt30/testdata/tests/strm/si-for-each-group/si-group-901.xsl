<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes=" xs"
    version="3.0">
   
  <!-- within a streaming template, use xsl:for-each-group group-adjacent, group on text nodes -->
  
  <xsl:mode name="s" streamable="yes"/>
       
    
  <xsl:template name="main">
    <xsl:source-document streamable="yes" href="../docs/books.xml">
    	<xsl:apply-templates select="/BOOKLIST/BOOKS" mode="s"/>
    </xsl:source-document>
  </xsl:template> 


  <xsl:template match="BOOKS" mode="s">
     <out>
      <xsl:for-each-group select="ITEM"
         group-adjacent="PRICE/text()">
         <group price="{current-grouping-key()}" count="{count(current-group()/..)}"/>
      </xsl:for-each-group> 
    </out>
  </xsl:template>   
       
</xsl:transform>

