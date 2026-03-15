<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:s="http://schema070.uri/"
        exclude-result-prefixes="xs">

<!-- PURPOSE: Test that failure to validate an attribute may be fatal even if the attribute
     is subsequently discarded because another attribute has the same name (Erratum E10) -->
     
<?spec xslt#validating-using-validation-attribute?>
<?spec xslt#annotation-for-constructed-attribute?>

<xsl:import-schema schema-location="schema070.xsd" 
     namespace="http://schema070.uri/"/>

<xsl:param name="dob1" select="'s:dob'"/>  
<xsl:param name="dob2" select="'s:dob'"/>  



  <xsl:template match="/" name="main">
    <s:out validation="strict">
      <xsl:attribute name="{$dob1}" select="'1951-02-29'" validation="strict"/>
      <xsl:attribute name="s:tob" select="'05:15:00'" validation="lax"/>
      <xsl:attribute name="pling" select="'05:15:00'" validation="lax"/> 
      <xsl:attribute name="plong" select="'05:15:00'" validation="strip"/>
      <xsl:attribute name="plang" select="'05:15:00'" validation="preserve"/>
      <xsl:attribute name="{$dob2}" select="'1951-10-11'" validation="strict"/>            
    </s:out>
  </xsl:template>
  
</xsl:stylesheet>