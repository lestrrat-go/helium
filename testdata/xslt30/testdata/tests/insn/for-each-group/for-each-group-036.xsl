<?xml version="1.0"?>
<?spec xslt#grouping?>
<xsl:stylesheet version="2.0"
                 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
   <!-- Test position() and last() with the result of distinct-values() -->
   <?category distinct-values?>              
   <xsl:template match="/">
     <xsl:variable name="languages">
       <lang>English</lang>
       <lang>English</lang>
       <lang>Chinese</lang>
       <lang>French</lang>
       <lang>French</lang>
       <lang>Chinese</lang>
       <lang>French</lang>
       <lang>Chinese</lang>
     </xsl:variable>
     <r>
     <xsl:for-each select="distinct-values($languages/lang)">
       <p><xsl:value-of select="position()"/></p>
       <v><xsl:value-of select="."/></v>
       <l><xsl:value-of select="last()"/></l>
     </xsl:for-each>
     </r>
   </xsl:template>
</xsl:stylesheet>

