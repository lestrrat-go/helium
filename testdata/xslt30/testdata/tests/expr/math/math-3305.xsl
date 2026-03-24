<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">
<?spec fo#func-abs?>
<!-- PURPOSE:  Test of abs() on all numeric data types. -->

<xsl:template match="doc">
<out>
<int a="{abs(3)}" b="{abs(-3)}" c="{abs(0)}"/>
<decimal a="{abs(3.2)}" b="{abs(-3.2)}" c="{abs(0.0)}"/>
<double a="{abs(3.2e0)}" b="{abs(-3.2e0)}" c="{abs(0.0e0)}" d="{abs(-0.0e0)}" e="{abs(number('NaN'))}"
   f="{abs(1e0 div 0)}" g="{abs(-1e0 div 0)}"/>
<float a="{abs(xs:float(3.2e0))}" b="{abs(xs:float(-3.2e0))}" c="{abs(xs:float(0.0e0))}" 
d="{abs(xs:float(-0.0e0))}" e="{abs(xs:float('NaN'))}"   f="{abs(xs:float(1e0 div 0))}" 
g="{abs(xs:float(-1e0 div 0))}"/>
</out>
</xsl:template>
</xsl:stylesheet>