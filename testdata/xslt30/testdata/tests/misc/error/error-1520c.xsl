<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if the
                           value of the type attribute of an
                              xsl:element, xsl:attribute,
                              xsl:copy, xsl:copy-of,
                              xsl:document, or xsl:result-document
                           instruction, or the xsl:type attribute of a literal result
                           element, is not a valid QName, or if it uses a prefix that
                           is not defined in an in-scope namespace declaration, or if the QName is
                           not the name of a type definition included in the in-scope schema
                              components for the stylesheet.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE1520?>

  <xsl:import-schema namespace="http://err1520b.com/">
      <xs:schema targetNamespace="http://err1520b.com/">
         <xs:element name="e" type="xs:date"/>
      </xs:schema>
  </xsl:import-schema>


  <xsl:template name="main">
    
      <out> 
         <e xsl:type="your::absent-type">2006-02-31</e>      
      </out>
  </xsl:template>
  




</xsl:stylesheet>
