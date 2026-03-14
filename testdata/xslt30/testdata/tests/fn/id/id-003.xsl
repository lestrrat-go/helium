<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Test use of id() in a pattern -->
<?spec xslt#patterns?>

  <xsl:template match="id('c')" priority="2">
    <id id="{@id}"/>
  </xsl:template>
  
  <xsl:template match="a" priority="1">
    <non-id id="{@id}"/>
  </xsl:template>
  
  <xsl:template match="/">
    <out><xsl:apply-templates/></out>
  </xsl:template>  
 
</xsl:stylesheet>
