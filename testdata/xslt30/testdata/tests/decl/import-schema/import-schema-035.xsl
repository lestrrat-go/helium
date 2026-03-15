<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:enu="http://schema044.uri/"
        exclude-result-prefixes="xs enu">

<!-- PURPOSE: Test use of union type -->
<?spec xslt#validation-xsl-type?>
<?spec xslt#annotation-for-constructed-attribute?>

<xsl:import-schema schema-location="schema044.xsd" 
     namespace="http://schema044.uri/"/>
  



  <xsl:template match="/">
    <out>
      <a><xsl:attribute name="list" select="'xml'" type="enu:method"/></a>
      <a><xsl:attribute name="list" select="'html'" type="enu:method"/></a>   
      <a><xsl:attribute name="list" select="10" type="enu:method"/></a>
      <a><xsl:attribute name="list" select="20" type="enu:method"/></a>                                           
    </out>
  </xsl:template>
  
</xsl:stylesheet>