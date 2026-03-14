<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:xs="http://www.w3.org/2001/XMLSchema"
        xmlns:s="http://schema071.uri/"
        exclude-result-prefixes="xs"
        default-validation="preserve">

<!-- PURPOSE: Test namespace fixup of QName-valued attributes.
       Now an error: parentless attributes of type QName no longer allowed -->
       <?spec xslt#annotation-for-constructed-attribute?>
       <?error XTSE1545?>

<xsl:import-schema schema-location="schema071.xsd" 
     namespace="http://schema071.uri/"/>
  



<xsl:variable name="att" as="attribute()">
  <xsl:attribute name="s:dob" select="'ab:value'" type="xs:QName" xmlns:ab="http://some-ab.uri/"/>  
</xsl:variable>

  <xsl:template match="/" >
    <out>
      <xsl:copy-of select="$att" validation="preserve"/>      
    </out>
  </xsl:template>
  
</xsl:stylesheet>