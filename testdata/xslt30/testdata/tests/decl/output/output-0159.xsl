<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <t:output method="html" indent="no"/>
   <!-- Purpose: Test that whitespace is not added or removed from elements with 
  				local names pre, script, style, textarea; indent=no with HTML output method.-->

   <t:template match="/">
      <html>
         <head>
            <script type="text/javascript">document.write ("&lt;EM&gt;This will work&lt;\/EM&gt;")
      
      </script>
            <style>

        &lt; &gt; &amp;

      </style>
         </head>
         <body>
            <pre>
               <b> Save    all	space here</b>
            </pre>
            <textarea rows="2" cols="20">
The cat was playing in the garden.

Suddenly a dog showed up.....
	  </textarea>
         </body>
      </html>
   </t:template>
</t:transform>
