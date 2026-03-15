<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:enu="http://schema056.uri/"
        exclude-result-prefixes="xs enu">

<!-- PURPOSE: Test a complex type involving an xs:group  -->

    <?spec xslt#validation-xsl-type?>
     <?error ?>

<xsl:import-schema schema-location="schema056.xsd" 
     namespace="http://schema056.uri/"/>
  



  <xsl:template match="/">
    <enu:out>
    <z xsl:type="enu:single-choice"><enu:j/><enu:k/><enu:j/><enu:k/></z>
    <z xsl:type="enu:single-choice"><fred/></z>
    </enu:out>
  </xsl:template>
  
</xsl:stylesheet>