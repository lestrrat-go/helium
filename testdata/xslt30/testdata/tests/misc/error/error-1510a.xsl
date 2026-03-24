<?xml version="1.0" encoding="UTF-8"?>
<!--If the validation attribute of an
                              xsl:element, xsl:attribute,
                              xsl:copy, xsl:copy-of, or
                              xsl:result-document instruction, or the
                              xsl:validation attribute of a literal result element, has
                           the effective value strict, and schema validity assessment
                           concludes that the validity of the element or attribute is invalid or
                           unknown, a type error occurs. As with other type errors, the error
                              may be signaled statically if it can be detected
                           statically. -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                version="2.0">

<?error XTDE0485?>

<!-- this test was designed to produce error 0485, but it doesn't.
     See W3C bugzilla 3069 --> 

   <xsl:import-schema namespace="http://err0485.com/">
      <xs:schema targetNamespace="http://err0485.com/">
         <xs:element name="out">
            <xs:complexType>
               <xs:sequence/>
               <xs:attribute name="q1" type="xs:QName"/>
               <xs:attribute name="q2" type="xs:QName"/>
            </xs:complexType>
         </xs:element>
      </xs:schema>
   </xsl:import-schema>  



  <xsl:template name="main">
      <out xmlns="http://err0485.com/" xsl:validation="strict">
         <xsl:attribute xmlns:a="http://one.uri/" name="q1">a:one</xsl:attribute>
         <xsl:attribute xmlns:a="http://two.uri/" name="q2">a:two</xsl:attribute>
      </out>
  </xsl:template>
  


</xsl:stylesheet>
