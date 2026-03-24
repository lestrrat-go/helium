<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.my.com"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Test case to show that circularity causes errors only if the parameter invloved in circulaity
              is actually evaluated at some point in the transformation process. -->

   <t:variable name="x" select="my:func(1)"/>

   <t:function name="my:func">
  		  <t:param name="a"/>
  		  <t:variable name="b" select="$x"/>  
  		  <t:sequence select="$a + 2"/>
	  </t:function>

   <t:template match="/doc">
		    <out>value of $x= <t:value-of select="$x"/>
      </out>	
	  </t:template>
</t:transform>
