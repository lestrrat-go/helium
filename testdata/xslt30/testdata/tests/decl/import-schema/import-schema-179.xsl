<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:local="http://localhost/ns/yes-no"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                exclude-result-prefixes="local xs"
                version="2.0">


  <!-- Purpose: Simple test of inline schema for a user defined simple 
  				type that is used as the type of a variable in the stylesheet. 
  				Example from Spec.-->
 
   <xsl:import-schema>
      <xs:schema targetNamespace="http://localhost/ns/yes-no">
         <xs:simpleType name="yes-no">
            <xs:restriction base="xs:string">
               <xs:enumeration value="yes"/>
               <xs:enumeration value="no"/>
            </xs:restriction>
         </xs:simpleType>
      </xs:schema>
   </xsl:import-schema>

   <xsl:variable name="condition" select="local:yes-no('yes')" as="local:yes-no"/>


   <xsl:template match="/doc">
      <out>
         <xsl:value-of select="data($condition) instance of local:yes-no "/>
         <xsl:value-of select="$condition"/>
      </out>
   </xsl:template>



</xsl:stylesheet>
