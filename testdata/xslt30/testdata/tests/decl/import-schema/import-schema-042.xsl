<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:enu="http://schema052.uri/"
        exclude-result-prefixes="xs enu">

<!-- PURPOSE: Test a complex type involving a repeatable choice -->
              <?spec xslt#import-schema?>
              <?spec xslt#validation-xsl-type?>

<xsl:import-schema schema-location="schema052.xsd" 
     namespace="http://schema052.uri/"/>
  



  <xsl:template match="/">
    <enu:out>
    <z xsl:type="enu:repeated-choice"><enu:p/></z>
    <z xsl:type="enu:repeated-choice"><enu:q/></z>
    <z xsl:type="enu:repeated-choice"><enu:r/></z>
    <z xsl:type="enu:repeated-choice"><enu:p/><enu:p/><enu:q/><enu:p/><enu:r/><enu:q/></z>
    </enu:out>
  </xsl:template>
  
</xsl:stylesheet>