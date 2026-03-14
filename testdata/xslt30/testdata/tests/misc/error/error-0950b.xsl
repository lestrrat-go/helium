<?xml version="1.0" encoding="UTF-8"?>
<!--It is a type error to use the
                           xsl:copy or xsl:copy-of instruction to
                        copy a node that has namespace-sensitive content if the
                           copy-namespaces attribute has the value no and
                        its explicit or implicit validation attribute has the value
                           preserve. It is also a type error if either of these
                        instructions (with validation="preserve") is used to copy an
                        attribute having namespace-sensitive content, unless the parent element is
                        also copied. A node has namespace-sensitive content if its typed value
                        contains an item of type xs:QName or xs:NOTATION
                        or a type derived therefrom. The reason this is an error is because the
                        validity of the content depends on the namespace context being
                        preserved.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTTE0950?>

   <xsl:import-schema namespace="http://err0950a.com/">
      <xs:schema targetNamespace="http://err0950a.com/">
         <xs:element name="e">
            <xs:complexType>
               <xs:sequence/>
               <xs:attribute name="q1" type="xs:QName"/>
            </xs:complexType>
         </xs:element>
      </xs:schema>
   </xsl:import-schema>

   <xsl:param name="p" select="''"/>

  <xsl:template xmlns="http://err0950a.com/" name="main">
      <xsl:variable name="x" as="element()">
         <e q1="my:value" xsl:validation="strict"/>
      </xsl:variable>
      <out>
        <xsl:copy-of select="$x/@q1" validation="preserve"/>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
