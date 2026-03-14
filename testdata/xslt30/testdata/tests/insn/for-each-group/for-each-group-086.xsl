<?xml version="1.0"?>
<?spec xslt#grouping?>
<xsl:stylesheet version="2.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  exclude-result-prefixes="xs">
   <!-- Test position() and last() with the result of xsl:for-each-group group-adjacent with
     a case-blind collation --> 
  
  <xsl:param name="collation" as="xs:string" select="'http://www.w3.org/2013/collation/UCA?strength=secondary'"/>
  
   <xsl:template match="/">
     <xsl:variable name="languages">
       <lang>English</lang>
       <lang>english</lang>
       <lang>Chinese</lang>
       <lang>French</lang>
       <lang>FRENCH</lang>
       <lang>Chinese</lang>
       <lang>French</lang>
       <lang>Chinese</lang>
     </xsl:variable>
     <r>
       <xsl:for-each-group select="$languages/lang" group-adjacent="." collation="{$collation}">
       <p><xsl:value-of select="position()"/></p>
       <v><xsl:value-of select="."/></v>
       <l><xsl:value-of select="last()"/></l>
     </xsl:for-each-group>
     </r>
   </xsl:template>
</xsl:stylesheet>

