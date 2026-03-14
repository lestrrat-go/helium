<?xml version="1.0"?> 
<t:stylesheet xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
  <!-- Test with two named output declarations with the same name and the same import 
    precedence and different values for the same attributes, then a third output declaration 
    with the same name but with higher precedence which sets values for the conflicting attributes. 
    Show that there is no conflict and the third named output declaration is used.-->
  
  
  <t:import href="output-0175b.xsl" />
  <t:import href="output-0175a.xsl" />
  
  <t:output method="text" encoding="UTF-8"/>
  <t:output method="xml" name="temp" cdata-section-elements="item1" indent="no" 
            include-content-type="yes" standalone="yes" encoding="UTF-8"/>
  
  <t:template match="/">
    <t:result-document format="temp">
      <out>
        <item1>A &amp; B</item1>
        <item2>A &amp; B</item2>
      </out>
    </t:result-document>
  </t:template>
  
</t:stylesheet>