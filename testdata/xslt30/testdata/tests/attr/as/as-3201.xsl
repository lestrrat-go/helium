<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:sg="http://www.example.com/ns/sub-group"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="sg"
             version="2.0">

<!--            Test type of global xsl:variable with @select selecting a typed element node from input file, 
  				@as="schema-element($QName)". Verify that selected element is annotated as an instance of the 
  				type defined by the schema element declaration $QName and its name comes from the substitution 
  				group of $QName.-->

   <t:import-schema namespace="http://www.example.com/ns/sub-group"
                    schema-location="subgroupTypeSchemaAs.xsd"/>

   <t:variable name="v"
               select="/sg:doc/sg:audio-book"
               as="schema-element(sg:book)"/>

   <t:template match="/">
      <out>
         <var1>
            <t:value-of select="$v instance of schema-element(sg:book)"/>
            <t:value-of select="$v instance of schema-element(sg:audio-book)"/>
         </var1>
      </out>
   </t:template>
</t:transform>
