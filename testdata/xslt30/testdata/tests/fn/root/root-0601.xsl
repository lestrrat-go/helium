<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Function is invoked with node(), element(), attribute() and comment() as arguments -->

   <t:template match="/">
	 	   <out>
         <t:value-of select="root( element() )"/>
         <t:value-of select="root( comment() )"/>
         <t:value-of select="root( node() )"/>
         <t:value-of select="root( text() )"/>
      </out>
	  </t:template>
</t:transform>
