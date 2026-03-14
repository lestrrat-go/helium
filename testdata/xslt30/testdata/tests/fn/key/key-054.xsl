<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- test key() function with a node-set argument  -->
<?spec xslt#key?>

  <xsl:key name="k" match="a" use="@id"/>
  <xsl:template match="/">
    <list><xsl:value-of select="key('k',//b/@ref)/@name" separator=","/></list>
  </xsl:template>
 
</xsl:stylesheet>
