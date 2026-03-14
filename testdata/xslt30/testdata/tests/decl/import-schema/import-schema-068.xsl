<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        xmlns:z="http://schema061.uri/"
        exclude-result-prefixes="xsi">

<!-- PURPOSE: Type-checking of constructed elements involving substitution groups -->

<xsl:import-schema namespace="http://schema061.uri/" schema-location="schema061.xsd"/>
<?spec xslt#import-schema?>
<?spec xpath#id-schema-element-test?>



  <xsl:template match="/">
    <out>
      <xsl:call-template name="p"/>
    </out>
  </xsl:template>
  
  <xsl:template name="p" as="schema-element(z:person)">
    <z:person xsl:validation="strict">
      <z:first>Michael</z:first>
      <z:middle>H</z:middle>
      <xsl:call-template name="sn"/>
    </z:person>
  </xsl:template>
  
  <xsl:template name="sn" as="schema-element(z:last)">
    <z:surname xsl:validation="strict">Kay</z:surname>
  </xsl:template>
  
</xsl:stylesheet>