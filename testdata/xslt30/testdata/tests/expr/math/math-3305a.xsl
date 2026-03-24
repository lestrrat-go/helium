<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">
<?spec fo#func-abs?>
<!-- PURPOSE:  Test of abs($p*) on all numeric data types. -->
  <!-- Variant forcing runtime evaluation -->
  
  <xsl:param name="p" select="1"/>

<xsl:template match="doc">
<out>;
<int a="{abs($p*3)}" b="{abs($p*-3)}" c="{abs($p*0)}"/>
<decimal a="{abs($p*3.2)}" b="{abs($p*-3.2)}" c="{abs($p*0.0)}"/>
<double a="{abs($p*3.2e0)}" b="{abs($p*-3.2e0)}" c="{abs($p*0.0e0)}" d="{abs($p*-0.0e0)}" e="{abs($p*number('NaN'))}"
   f="{abs($p*1e0 div 0)}" g="{abs($p*-1e0 div 0)}"/>
<float a="{abs($p*xs:float(3.2e0))}" b="{abs($p*xs:float(-3.2e0))}" c="{abs($p*xs:float(0.0e0))}" 
d="{abs($p*xs:float(-0.0e0))}" e="{abs($p*xs:float('NaN'))}"   f="{abs($p*xs:float(1e0 div 0))}" 
g="{abs($p*xs:float(-1e0 div 0))}"/>
</out>
</xsl:template>
</xsl:stylesheet>