<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:s="http://schema071.uri/"
        exclude-result-prefixes="xs"
        default-validation="preserve">

<!-- PURPOSE: Test validation of QName-valued elements and attributes 
      This is now an error (XTSE1545) -->
      
<?spec xslt#annotation-for-constructed-attribute?>
       <?error XTSE1545?>

<xsl:import-schema schema-location="schema071.xsd" 
     namespace="http://schema071.uri/"/>
  



  <xsl:template match="/" xmlns:ab="http://some-ab.uri/">
    <out>
      <xsl:attribute name="s:dob" select="'ab:value'" type="xs:QName"/>
      <xsl:attribute name="s:tob" select="'ab:value'" type="s:subQName"/>
      <xsl:attribute name="s:the-qname" select="'ab:luvue'" validation="strict"/>
      <xsl:attribute name="pling" select="'value'" type="xs:QName"/>
      <xsl:attribute name="plong" select="'value'" type="xs:QName"/>        
    </out>
  </xsl:template>
  
</xsl:stylesheet>