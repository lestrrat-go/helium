<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://ns.example.com"
             xmlns:one="http://ns.example.com"
             xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my one"
             version="2.0">

   <t:output method="xhtml"
             cdata-section-elements="h1 my:h3 h5"
             indent="no"
             encoding="UTF-8"/>
   <!-- Purpose: Test of XHTML output with @cdata-section-elements 
  				with a list of names with or without prefixes.-->

   <t:template match="/">
      <html>
         <t:text>
</t:text>
         <head>
            <title/>
         </head>
         <t:text>
</t:text>
         <body>
            <t:text>
</t:text>
            <h1>a &amp; b</h1>
            <t:text>
</t:text>
            <h2>a &amp; b</h2>
            <t:text>
</t:text>
            <h3>a &amp; b</h3>
            <t:text>
</t:text>
            <one:h3>a &amp; b</one:h3>
            <t:text>
</t:text>
            <my:h3>a &amp; b</my:h3>
            <t:text>
</t:text>
            <h3 xmlns="http://www.mytest.example.org">a &amp; b</h3>
            <t:text>
</t:text>
            <h4>a &amp; b</h4>
            <t:text>
</t:text>
            <h5>a &amp; b</h5>
            <t:text>
</t:text>
         </body>
      </html>
   </t:template>
</t:transform>
