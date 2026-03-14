<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:s="http://schema070.uri/"
        exclude-result-prefixes="xs">

<!-- PURPOSE: Test validation of attributes -->
<?spec xslt#validating-using-validation-attribute?>
<?spec xslt#annotation-for-constructed-attribute?>

<xsl:import-schema schema-location="schema070.xsd" 
     namespace="http://schema070.uri/"/>
  



  <xsl:template match="/">
    <s:out>
      <xsl:attribute name="s:dob" select="'1951-10-11'" validation="strict"/>
      <xsl:attribute name="s:tob" select="'05:15:00'" validation="lax"/>
      <xsl:attribute name="pling" select="'05:15:00'" validation="lax"/> 
      <xsl:attribute name="plong" select="'05:15:00'" validation="strip"/>
      <xsl:attribute name="plang" select="'05:15:00'" validation="preserve"/>            
    </s:out>
  </xsl:template>
  
</xsl:stylesheet>