<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

                       <!--It is a type error if the
                           value of the type attribute of an
                           xsl:copy or xsl:copy-of instruction 
                           refers to a complex type definition and one or more of the items being copied
                           is an attribute node.-->
                           
   <xsl:import-schema namespace="http://err1535.com/">
      <xs:schema targetNamespace="http://err1535.com/">
         <xs:complexType name="c">
           <xs:sequence>
             <xs:element name="e"/>
           </xs:sequence>
         </xs:complexType>
      </xs:schema>
   </xsl:import-schema>
   
   <xsl:variable name="v"><a code="23"/></xsl:variable>
   
   <xsl:template name="main">
      <out>
         <xsl:copy select="$v//@code" type="Q{http://err1535.com/}c"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
