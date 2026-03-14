<?xml version="1.0" encoding="UTF-8"?>
<!--If the validation attribute of an
                              xsl:element, xsl:attribute,
                              xsl:copy, xsl:copy-of, or
                              xsl:result-document instruction, or the
                              xsl:validation attribute of a literal result element, has
                           the effective value lax, and schema validity assessment
                           concludes that the element or attribute is invalid, a type error occurs.
                           As with other type errors, the error may be signaled
                           statically if it can be detected statically. -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTTE1512?>

  <xsl:import-schema namespace="http://err1515.com/">
      <xs:schema targetNamespace="http://err1515.com/">
         <xs:element name="e" type="xs:date"/>
      </xs:schema>
  </xsl:import-schema>


  <xsl:template name="main">
    
      <out> 
         <xsl:element name="e" namespace="http://err1515.com/" validation="lax">2006-02-31</xsl:element>      
      </out>
  </xsl:template>
  




</xsl:stylesheet>
