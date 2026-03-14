<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" version="2.0">
   
   <xsl:import-schema>
       <xs:schema>
           <xs:element name="list">
               <xs:complexType>
                   <xs:sequence>
                       <xs:element ref="element" minOccurs="0" maxOccurs="unbounded"/>
                   </xs:sequence>
               </xs:complexType>
           </xs:element>
           
           <xs:element name="element" type="type1">
               <xs:alternative test="@type = 'type2'" type="type2"/>
           </xs:element>
           
           <xs:complexType name="type1">
               <xs:sequence>
                   <xs:element name="child1"/>
               </xs:sequence>
               <xs:attribute name="type" type="xs:string"/>
           </xs:complexType>
           
           <xs:complexType name="type2">
               <xs:complexContent>
                   <xs:extension base="type1">
                       <xs:sequence>
                           <xs:element name="child2"/>
                       </xs:sequence>
                   </xs:extension>
               </xs:complexContent>
           </xs:complexType>
       </xs:schema>
   </xsl:import-schema>
   
   <xsl:template match="/" name="main">
       <list xsl:validation="strict"> <!-- strict validation here works -->
           <element xsl:validation="strict"> <!-- strict validation here causes error -->
               <xsl:attribute name="type" select="'type2'"/>
               <child1/>
               <child2/>
           </element>
       </list>
   </xsl:template>
   
</xsl:transform> 