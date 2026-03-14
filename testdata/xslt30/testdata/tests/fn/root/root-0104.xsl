<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:test="http://www.example.com/example"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0"
             exclude-result-prefixes="test">
<!-- Purpose: Function is invoked with element node in variable -->

   <t:template match="/">
	 	   <out>
         <t:variable name="node1" select="doc/test:a"/>
         <t:variable name="node2" select="doc"/>
         <t:value-of select="root( $node1 )"/>
         <t:value-of select="root( $node2 )"/>
      </out>
	  </t:template>
</t:transform>
