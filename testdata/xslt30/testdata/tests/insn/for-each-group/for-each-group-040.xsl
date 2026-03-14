<?xml version="1.0"?>
<?spec xslt#grouping?>
<xsl:stylesheet version="2.0"
                 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
   <!-- Test position() and last() with the result of xsl:for-each-group group-ending-with -->              
   <xsl:template match="/">
     <xsl:variable name="languages">
       <p>English</p>
       <lang>English</lang>
       <lang>Chinese</lang>
       <p>French</p>
       <lang>French</lang>
       <p>Chinese</p>
       <lang>French</lang>
       <lang>Chinese</lang>
     </xsl:variable>
     <r>
     <xsl:for-each-group select="reverse($languages/*)" group-ending-with="p">
       <p><xsl:value-of select="position()"/></p>
       <v><xsl:value-of select="."/></v>
       <l><xsl:value-of select="last()"/></l>
     </xsl:for-each-group>
     </r>
   </xsl:template>
</xsl:stylesheet>

