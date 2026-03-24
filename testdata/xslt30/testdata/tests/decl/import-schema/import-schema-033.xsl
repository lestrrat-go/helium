<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:enu="http://schema042.uri/"
        exclude-result-prefixes="xs enu">

<!-- PURPOSE: Test use of facets controlling length of lists -->
<?spec xslt#validation-xsl-type?>
<?spec xslt#annotation-for-constructed-attribute?>

<xsl:import-schema schema-location="schema042.xsd" 
     namespace="http://schema042.uri/"/>
  



  <xsl:template match="/">
    <out>
      <a><xsl:attribute name="list" select="1 to 10" type="enu:list10int"/></a>
      <a><xsl:attribute name="list" select="21 to 30" type="enu:list10int"/></a>   
      <a><xsl:attribute name="list" select="1 to 5" type="enu:list5-8int"/></a>
      <a><xsl:attribute name="list" select="21 to 28" type="enu:list5-8int"/></a>                         
    </out>
  </xsl:template>
  
</xsl:stylesheet>