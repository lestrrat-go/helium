<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:ul="http://www.example.com/ns/union-list"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs ul"
                version="2.0">
                
<!--            Test of xsl:template which contains a typed element node from input file, or explicitly created 
  				xsl:element or LRE with xsl:type, @as="element(QName, user-defined-union-type)". There is only 
  				one member type, one of: -built-in primitive, built-in derived, user-defined atomic, list-->

   <xslt:import-schema namespace="http://www.example.com/ns/union-list"
                       schema-location="list-unionSchemaAs.xsd"/>

   <xslt:template match="/">
      <out>
         
         <temp1>
            <xslt:call-template name="t1"/>
         </temp1>
         
         <temp2>
            <xslt:call-template name="t2"/>
         </temp2>
         
         <temp3>
            <xslt:call-template name="t3"/>
         </temp3>
         
         <temp4>
            <xslt:call-template name="t4"/>
         </temp4>
         
         <temp5>
            <xslt:call-template name="t5"/>
         </temp5>
         
         <temp6>
            <xslt:call-template name="t6"/>
         </temp6>
         
         <temp7>
            <xslt:call-template name="t7"/>
         </temp7>
         
         <temp8>
            <xslt:call-template name="t8"/>
         </temp8>
         
         <temp9>
            <xslt:call-template name="t9"/>
         </temp9>
         
         <temp10>
            <xslt:call-template name="t10"/>
         </temp10>
         
         <temp11>
            <xslt:call-template name="t11"/>
         </temp11>
         
         <temp12>
            <xslt:call-template name="t12"/>
         </temp12>
      </out>
   </xslt:template>

   <xslt:template name="t1" as="element(ul:union-primitive, ul:primitiveUnionType)">
	     <xslt:copy-of select="/ul:userNode/ul:union-primitive[1]" validation="strict"/>
   </xslt:template>

   <xslt:template name="t2" as="element(ul:union-primitive, ul:primitiveUnionType)">
	     <ul:union-primitive xslt:type="ul:primitiveUnionType">two words</ul:union-primitive>
   </xslt:template>

   <xslt:template name="t3" as="element(ul:union-derived, ul:derivedUnionType)">
	     <xslt:element name="ul:union-derived" type="ul:derivedUnionType">local-name</xslt:element>
   </xslt:template>

   <xslt:template name="t4" as="element(ul:union-derived, ul:derivedUnionType)">
	     <ul:union-derived xslt:type="ul:derivedUnionType">-128</ul:union-derived>
   </xslt:template>

   <xslt:template name="t5"
                  as="element(ul:union-user-defined, ul:user-definedUnionType)">
	     <xslt:copy-of select="/ul:userNode/ul:union-user-defined[1]" validation="strict"/>
   </xslt:template>

   <xslt:template name="t6"
                  as="element(ul:union-user-defined, ul:user-definedUnionType)">
	     <ul:union-user-defined xslt:type="ul:user-definedUnionType">000-AB</ul:union-user-defined>
   </xslt:template>

   <xslt:template name="t7" as="element(ul:union-list, ul:listUnionType)">
	     <xslt:copy-of select="/ul:userNode/ul:union-list[3]" validation="strict"/>
   </xslt:template>

   <xslt:template name="t8" as="element(ul:union-list, ul:listUnionType)">
	     <ul:union-list xslt:type="ul:listUnionType">123-CD 456-EF</ul:union-list>
   </xslt:template>

   <xslt:template name="t9"
                  as="element(ul:part-integer-union, ul:partIntegerUnionType)">
	     <xslt:copy-of select="/ul:userNode/ul:part-integer-union[1]" validation="strict"/>
   </xslt:template>

   <xslt:template name="t10"
                  as="element(ul:part-integer-union, ul:partIntegerUnionType)">
	     <ul:part-integer-union xslt:type="ul:partIntegerUnionType">4</ul:part-integer-union>
   </xslt:template>

   <xslt:template name="t11" as="element(ul:union-complex, ul:complexUnionType)">
	     <xslt:copy-of select="/ul:userNode/ul:union-complex[1]" validation="strict"/>
   </xslt:template>

   <xslt:template name="t12" as="element(ul:union-complex, ul:complexUnionType)">
	     <ul:union-complex xslt:type="ul:complexUnionType">127</ul:union-complex>
   </xslt:template>
</xslt:transform>
