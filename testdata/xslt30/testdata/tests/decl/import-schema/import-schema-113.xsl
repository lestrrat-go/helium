<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns="http://schema061.uri/"
        exclude-result-prefixes="xs">

<!-- PURPOSE: Test validation="strict", including nested validation of child elements -->

<!-- error: age is non-numeric: static validation via a conditional -->
<!-- reports a type error if validated statically, but not if done dynamically with default param value -->

     <?spec xslt#validating-using-validation-attribute?>
<?error ?>

<xsl:import-schema schema-location="schema061.xsd" 
     namespace="http://schema061.uri/"/>
  



  <xsl:template match="/">
    <address zip="rg4" xsl:validation="strict">
      <person>
        <first>Michael</first>
        <middle>H</middle>
        <last>Kay</last>
      </person>
      <xsl:choose>
        <xsl:when test="$p = 1">
            <age><old/></age>
        </xsl:when>
        <xsl:otherwise>
            <age>52</age>
        </xsl:otherwise>
     </xsl:choose>
    </address>
  </xsl:template>
  
  <xsl:param name="p" select="1"/>
  
</xsl:stylesheet>