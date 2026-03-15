<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:cc="http://copy2195.com/"
    exclude-result-prefixes="xs">

  <!-- FileName: copy2195err -->
    <!-- Purpose: Use copy-of() to copy an attribute with namespace-sensitive content -->
  
<xsl:import-schema namespace="http://copy2195.com/">
    <xs:schema targetNamespace="http://copy2195.com/">
        <xs:element name="e">
            <xs:complexType>
                     <xs:attribute name="a" type="xs:QName"/>
             </xs:complexType>
        </xs:element>
    </xs:schema>
</xsl:import-schema>  


<xsl:template match="/">
  <xsl:variable name="e" as="schema-element(cc:e)">
    <cc:e a="cc:qname-value" xsl:validation="strict"/>
  </xsl:variable>  
  <out>
    <xsl:sequence select="copy-of($e/@*)"/>
  </out>
</xsl:template>

</xsl:stylesheet>