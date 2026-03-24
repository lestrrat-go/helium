<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:enu="http://schema046.uri/"
        exclude-result-prefixes="xs enu">

<!-- PURPOSE: Test use of a list type whose members are a union type -->
<?spec xslt#validation-xsl-type?>
<?spec xslt#annotation-for-constructed-attribute?>

<xsl:import-schema schema-location="schema046.xsd" 
     namespace="http://schema046.uri/"/>
  



  <xsl:template match="/">
    <out>
      <a><xsl:attribute name="list" select="'xml text'" type="enu:methods"/></a>
      <a><xsl:attribute name="list" select="'html 15 xml'" type="enu:methods"/></a>   
      <a><xsl:attribute name="list" select="(10, 20, 'html')" type="enu:methods"/></a>
      <a><xsl:attribute name="list" select="20" type="enu:methods"/></a>                                           
    </out>
  </xsl:template>
  
</xsl:stylesheet>