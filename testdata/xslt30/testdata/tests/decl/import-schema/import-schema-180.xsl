<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:foo="http://ns.example.com/sch002/025"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                exclude-result-prefixes="foo xs"
                version="2.0">


  <!-- Purpose: Test with xsl:import-schema without @namespace which 
  				contains an xs:schema element with @targetNamespace. 
  				Verify that elements defined in the targetNamespace 
  				from the imported schema are available for use in the stylesheet.-->
 
   <xsl:import-schema>
      <xs:schema targetNamespace="http://ns.example.com/sch002/025">
         <xs:element name="myTest" type="foo:testType"/>
         <xs:element name="complexTest" type="foo:complexTestType"/>
         <xs:attribute name="type" type="foo:testType"/>
         <xs:simpleType name="testType">
            <xs:restriction base="xs:positiveInteger">
               <xs:minInclusive value="1"/>
               <xs:maxInclusive value="10000"/>
            </xs:restriction>
         </xs:simpleType>
         <xs:complexType name="complexTestType">
            <xs:sequence>
               <xs:element name="subject" type="foo:subjectType"/>
               <xs:element name="size" type="xs:integer"/>
            </xs:sequence>
            <xs:attribute ref="foo:type"/>
         </xs:complexType>
         <xs:simpleType name="subjectType">
            <xs:restriction base="xs:string">
               <xs:enumeration value="math"/>
               <xs:enumeration value="language"/>
               <xs:enumeration value="chemistry"/>
               <xs:enumeration value="geography"/>
            </xs:restriction>
         </xs:simpleType>
      </xs:schema>
   </xsl:import-schema>

   <xsl:variable name="var1" as="foo:testType*">
      <xsl:sequence select="(foo:testType(2000),foo:testType(2001),foo:testType(2002),foo:testType(2003))"/>
   </xsl:variable>

   <xsl:variable name="var2" as="element()">
      <foo:complexTest xsl:type="foo:complexTestType" foo:type="333">
         <subject>math</subject>
         <size>50</size>
      </foo:complexTest>
   </xsl:variable>

   <xsl:variable name="var3" as="attribute()">
	     <xsl:attribute name="foo:type" type="foo:testType">444</xsl:attribute>
   </xsl:variable>

   <xsl:template match="/doc">
      <xsl:text>
</xsl:text>
      <out>
         <xsl:text>
</xsl:text>
         <type>
            <xsl:value-of select="sum($var1)"/>
         </type>
         <xsl:text>
</xsl:text>
         <elem>
            <xsl:value-of select="$var2 instance of schema-element(foo:complexTest)"/>
            <xsl:text> * </xsl:text>
            <xsl:value-of select="$var2/@foo:type instance of schema-attribute(foo:type)"/>
            <xsl:text> * </xsl:text>
            <xsl:copy-of select="$var2"/>
         </elem>
         <xsl:text>
</xsl:text>
         <attrib>
            <xsl:value-of select="$var3 instance of schema-attribute(foo:type)"/>
            <xsl:text> * </xsl:text>
            <xsl:value-of select="$var3"/>
         </attrib>
         <xsl:text>
</xsl:text>
      </out>
   </xsl:template>



</xsl:stylesheet>
