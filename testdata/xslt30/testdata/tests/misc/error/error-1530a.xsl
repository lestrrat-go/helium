<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if the
                           value of the type attribute of an
                              xsl:attribute instruction refers to a complex type
                           definition-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE1530?>

  <xsl:import-schema namespace="http://err1530.com/">
      <xs:schema targetNamespace="http://err1530.com/">
         <xs:complexType name="t">
            <xs:sequence>
               <xs:element name="e"/>
            </xs:sequence>
         </xs:complexType>
      </xs:schema>
  </xsl:import-schema>


  <xsl:template name="main">
    
      <out> 
         <e>
            <xsl:attribute name="d" type="t" xpath-default-namespace="http://err1530.com/">2006-02-31</xsl:attribute>
         </e>      
      </out>
  </xsl:template>
  




</xsl:stylesheet>
