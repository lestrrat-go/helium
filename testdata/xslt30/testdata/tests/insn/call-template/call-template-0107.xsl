<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:My="http://www.examples.com/NotMine"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="My"
             version="2.0">
<!-- Purpose: Launched with initial template but there is no template of that name, 
  				where the name is a QName and a different prefix is used.-->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/">
	     <out/>
   </t:template>

   <t:template name="My:temp">
	     <t:text> Do Nothing! </t:text>
   </t:template>
</t:transform>
