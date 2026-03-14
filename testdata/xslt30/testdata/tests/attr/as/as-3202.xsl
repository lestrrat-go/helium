<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:sg="http://www.example.com/ns/sub-group"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="sg"
             version="2.0">

<!--            Test type of result of xsl:function with @as="schema-element($QName)". Sequence constructor 
  				contains a typed element node from input file.Verify that selected element is annotated as an 
  				instance of the type defined by the schema element declaration $QName and its name comes from 
  				the substitution group of $QName.-->

   <t:import-schema namespace="http://www.example.com/ns/sub-group"
                    schema-location="subgroupTypeSchemaAs.xsd"/>

   <t:function name="sg:f1" as="schema-element(sg:book)">
	     <t:copy-of select="document('as-32b.xml')/sg:doc/sg:audio-book"
                 validation="strict"/>
   </t:function>

   <t:template match="/">
      <out>
         <t:value-of select="sg:f1() instance of schema-element(sg:book)"/>
         <t:value-of select="sg:f1() instance of schema-element(sg:audio-book)"/>
      </out>
   </t:template>
</t:transform>
