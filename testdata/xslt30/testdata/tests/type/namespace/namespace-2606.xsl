<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- test inherit-namespaces="no" on xsl:element -->


<xsl:output method="xml" version="1.1"  undeclare-prefixes="yes"/>

<xsl:template match = "/">
  <xsl:element name="a" inherit-namespaces="no">
    <xsl:namespace name="n">http://n1.com/</xsl:namespace>
    <xsl:namespace name="m">http://m1.com/</xsl:namespace>    
    <xsl:namespace name="k">http://k1.com/</xsl:namespace>        
    <xsl:element name="b" inherit-namespaces="no">
      <xsl:namespace name="n">http://n1.com/</xsl:namespace>
      <xsl:namespace name="k">http://k2.com/</xsl:namespace>      
      <xsl:element name="c"/>
    </xsl:element>
  </xsl:element>
</xsl:template>

</xsl:stylesheet>
