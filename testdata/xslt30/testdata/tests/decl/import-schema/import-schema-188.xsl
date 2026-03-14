<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:local="http://www.multImpTest.org/msimult1"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                exclude-result-prefixes="local xs"
                version="2.0">

  <!-- Purpose: Test with inline schema importing another schema.
  				The imported declarations are not used in the stylesheet.-->
 
   <xsl:import-schema namespace="http://www.multImpTest.org/msimult1">
      <schema xmlns="http://www.w3.org/2001/XMLSchema"
              xmlns:svimp="http://www.multImpTest.org/msimult1_imp"
              xmlns:sv="http://www.multImpTest.org/msimult1"
              targetNamespace="http://www.multImpTest.org/msimult1">
         <import namespace="http://www.multImpTest.org/msimult1_imp"
                 schemaLocation="msimult1-1.xsd"/>
         <import namespace="http://www.multImpTest.org/msimult1_imp"
                 schemaLocation="msimult1-2.xsd"/>
         <element name="elem5" type="string"/>
         <element name="root">
            <complexType>
               <sequence>
                  <element ref="local:elem5" maxOccurs="unbounded"/>
               </sequence>
            </complexType>
         </element>
      </schema>
   </xsl:import-schema>

   <xsl:variable name="var" as="element()">
	     <local:elem5 xsl:validation="strict">hello</local:elem5>
   </xsl:variable>

   <xsl:template match="/doc">
      <out>
         <xsl:value-of select="$var instance of element(*, xs:string)"/>
         <xsl:value-of select="$var instance of schema-element(local:elem5)"/>
         <xsl:value-of select="$var"/>
      </out>
   </xsl:template>


</xsl:stylesheet>
