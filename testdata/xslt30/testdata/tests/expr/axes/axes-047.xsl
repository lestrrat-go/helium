<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#axes?>
  <!-- PURPOSE: Test for repeated use of //name (Saxon optimisation). -->
<xsl:template match="/">
   <out>
	  <xsl:for-each select="//node()">
        <xsl:apply-templates select="//edge"/>;
      </xsl:for-each>
   </out>
</xsl:template>
  
<xsl:template match="edge">
   <xsl:value-of select="."/>
</xsl:template>
 
</xsl:stylesheet>
