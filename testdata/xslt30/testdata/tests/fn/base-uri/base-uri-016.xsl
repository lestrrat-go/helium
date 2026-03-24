<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
             xml:base="http://www.example.org/"
             version="2.0">
<!-- Purpose: Testcase with fn:base-uri of a variable. Relative URI is set in @xml:base on the variable instruction.-->

   <t:template match="/">
	     <out>
         <t:variable name="elemcopy" xml:base="/main/">
		hello
	</t:variable>
         <t:value-of select="base-uri($elemcopy)"/>
      </out>
   </t:template>
</t:transform>
