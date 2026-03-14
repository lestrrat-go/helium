<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:s="http://schema071.uri/"
        exclude-result-prefixes="xs"
        default-validation="preserve">

<!-- PURPOSE: error condition: element has two QName valued attributes using the same prefix to refer to different URIs -->
<?spec xslt#annotation-for-constructed-attribute?>
       <?error XTSE1545?>

<xsl:import-schema schema-location="schema071.xsd" 
     namespace="http://schema071.uri/"/>
  



  <xsl:template match="/" xmlns:ab="http://some-ab.uri/">
    <out>
      <xsl:attribute name="s:dob" select="'ab:value'" type="xs:QName"/>
      <xsl:attribute name="s:tob" select="'ab:value'" type="s:subQName" xmlns:ab="http://some-other.uri/"/>
    </out>
  </xsl:template>
  
</xsl:stylesheet>