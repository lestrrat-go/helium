<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.example.com/ns/match/attributes"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:t="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="my xs"
                version="2.0">
<!--        Test of template matching with schema-attribute($name) for top-level attribute declarations 
				of derived built-in atomic types. Show all cases where a node will match the pattern-->

   <t:import-schema namespace="http://www.example.com/ns/match/attributes"
                       schema-location="match-attributes.xsd"/>

   <t:template match="/my:people">     
	   <out>
         <t:apply-templates select="//@*"/>
      </out>
   </t:template>

   <t:template match="schema-attribute(my:nationality)">
	     <t:element name="A">
		       <t:value-of select="."/>
	     </t:element>     
   </t:template>

   <t:template match="schema-attribute(my:id)">
	     <t:element name="B">
		       <t:value-of select="."/>
	     </t:element>   
   </t:template>

   <t:template match="schema-attribute(my:ref)">
	     <t:element name="C">
		       <t:value-of select="."/>
	     </t:element> 
   </t:template>

   <t:template match="schema-attribute(my:refs)">
	     <t:element name="D">
		       <t:value-of select="."/>
	     </t:element>   
   </t:template>

   <t:template match="schema-attribute(my:tok)">
	     <t:element name="E">
		       <t:value-of select="."/>
	     </t:element>   
   </t:template>

   <t:template match="schema-attribute(my:toks)">
	     <t:element name="F">
		       <t:value-of select="."/>
	     </t:element>    
   </t:template>

   <t:template match="schema-attribute(my:language)">
	     <t:element name="G">
		       <t:value-of select="."/>
	     </t:element>     
   </t:template>

   <t:template match="schema-attribute(my:positiveInteger)">
	     <t:element name="H">
		       <t:value-of select="."/>
	     </t:element>    
   </t:template>

   <t:template match="schema-attribute(my:unsignedShort)">
	     <t:element name="I">
		       <t:value-of select="."/>
	     </t:element>    
   </t:template>

   <t:template match="schema-attribute(my:long)">
	     <t:element name="J">
		       <t:value-of select="."/>
	     </t:element>   
   </t:template>

   <t:template match="attribute(*)">
	     <wrong>
         <t:value-of select="."/>
      </wrong>	     
   </t:template>
</t:transform>
