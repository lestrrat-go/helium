<?xml version="1.0"?>
<t:stylesheet xmlns:t="http://www.w3.org/1999/XSL/Transform" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  exclude-result-prefixes="xs" version="2.0">
  
  <t:template match="element(*, xs:untyped)">
    <b><t:next-match /></b>
  </t:template>
  
</t:stylesheet>