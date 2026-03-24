<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#axes?>
  <!-- PURPOSE: Test for ancestor-or-self::*[@att1][1]/@att1 vs. (ancestor-or-self::*)[@att1][1]/@att1. -->
<xsl:template match="/">
   <out>
      <xsl:for-each select="//baz">
         <xsl:value-of select="ancestor-or-self::*[@att1][1]/@att1"/><xsl:text>, </xsl:text>
         <xsl:value-of select="(ancestor-or-self::*)[@att1][1]/@att1"/>
      </xsl:for-each>
   </out>
</xsl:template>
 
</xsl:stylesheet>
